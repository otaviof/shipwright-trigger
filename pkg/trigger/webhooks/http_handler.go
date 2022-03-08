package webhooks

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/otaviof/shipwright-trigger/pkg/trigger/inventory"
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildclientset "github.com/shipwright-io/build/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// HTTPHandler reprents the webhook endpoint, which parses the events, uses the Inventory to find the
// respective Build instances. At the end, fires BuildRuns using the informed client.
type HTTPHandler struct {
	ctx context.Context

	webHookEventHandler Interface                // provider specific event handler interface
	buildInventory      inventory.Interface      // build inventory instance
	buildClientset      buildclientset.Interface // shipwright clientset
	clientset           kubernetes.Interface     // kubernetes clientset
	secretKeyName       string
}

// createBuildRun creates a BuildRun object for the informed Build, the BuildRun name is based on
// Kubernetes generated name.
func (h *HTTPHandler) createBuildRun(buildName types.NamespacedName) error {
	log.Printf("Creating a BuildRun for the %q Build", buildName.String())
	br := &v1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", buildName.Name),
		},
		Spec: v1alpha1.BuildRunSpec{
			BuildRef: v1alpha1.BuildRef{
				Name: buildName.Name,
			},
		},
	}
	var err error
	br, err = h.buildClientset.ShipwrightV1alpha1().
		BuildRuns(buildName.Namespace).
		Create(h.ctx, br, metav1.CreateOptions{})
	if err == nil {
		log.Printf("BuildRun '%s/%s' created for the %q Build",
			br.GetNamespace(), br.GetName(), buildName.String())
	}
	return err
}

// validateSecretToken it retrieves the secret and extract the token for the payload validation.
func (h *HTTPHandler) validateSecretToken(rp *RequestPayload, secretName types.NamespacedName) error {
	secret, err := h.clientset.CoreV1().
		Secrets(secretName.Namespace).
		Get(h.ctx, secretName.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	token, ok := secret.Data[h.secretKeyName]
	if !ok {
		return fmt.Errorf("unable to find key %q on secret %q payload", h.secretKeyName, secretName)
	}
	return h.webHookEventHandler.ValidateSignature(rp, token)
}

// dispatch genereate a BuildRun object based on the informed selector after validating the payload
// against it signature and secret.
func (h *HTTPHandler) dispatch(rp *RequestPayload, selector *BuildSelector) error {
	log.Printf("Searching Builds for %q repository on revision %q",
		selector.RepoURL, selector.Revision)
	builds := h.buildInventory.SearchForGit(selector.WhenType, selector.RepoURL, selector.Revision)
	for _, result := range builds {
		if result.HasSecret() {
			log.Printf("Validating request for Build %q against %q secret",
				result.BuildName, result.SecretName)
			if err := h.validateSecretToken(rp, result.SecretName); err != nil {
				return err
			}
			log.Print("Payload validated successfully against secret token!")
		}
		if err := h.createBuildRun(result.BuildName); err != nil {
			return err
		}
	}
	return nil
}

// handleWebHookEvent parses the informed event in order to extract a BuildSelector and more request
// information to validate the secret and signature later on.
func (h *HTTPHandler) handleWebHookEvent(r *http.Request) error {
	rp, err := h.webHookEventHandler.ExtractRequestPayload(r)
	if err != nil {
		return nil
	}

	selector, err := h.webHookEventHandler.ExtractBuildSelector(rp)
	if err != nil {
		return nil
	}
	if selector.IsEmpty() {
		return nil
	}

	return h.dispatch(rp, selector)
}

// HandleRequest webhook primary endpoint, replies empty payload when successful, and shares the
// error message otherwise.
func (h *HTTPHandler) HandleRequest(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-type", "application/json")
	if err := h.handleWebHookEvent(r); err != nil {
		log.Printf("Error processing the webhook request: %q", err)
		rw.WriteHeader(http.StatusInternalServerError)
		io.WriteString(rw, fmt.Sprintf("{ \"error\": %q }", err))
	} else {
		rw.WriteHeader(http.StatusOK)
		io.WriteString(rw, "{}")
	}
}

func NewHTTPHandler(
	ctx context.Context,
	webHookEventHandler Interface,
	buildInventory inventory.Interface,
	buildClientset buildclientset.Interface,
	clientset kubernetes.Interface,
	secretKeyName string,
) *HTTPHandler {
	return &HTTPHandler{
		ctx:                 ctx,
		webHookEventHandler: webHookEventHandler,
		buildInventory:      buildInventory,
		buildClientset:      buildClientset,
		clientset:           clientset,
		secretKeyName:       secretKeyName,
	}
}
