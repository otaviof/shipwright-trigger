package webhooks

import (
	"context"
	"net/http"

	"github.com/otaviof/shipwright-trigger/pkg/trigger/clients"
	"github.com/otaviof/shipwright-trigger/pkg/trigger/inventory"
	buildclientset "github.com/shipwright-io/build/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

type HTTPServer struct {
	ctx            context.Context
	buildInventory inventory.Interface
	buildClientset buildclientset.Interface // shipwright clientset
	clientset      kubernetes.Interface     // kubernetes clientset
}

const (
	GitHubSecretKeyName  = "github-token"
	GitHubWebHookPattern = "/"
)

func (s *HTTPServer) Listen(addr string) error {
	githubHandler := NewHTTPHandler(
		s.ctx,
		NewGitHubWebHook(),
		s.buildInventory,
		s.buildClientset,
		s.clientset,
		GitHubSecretKeyName,
	)
	http.HandleFunc(GitHubWebHookPattern, githubHandler.HandleRequest)

	return http.ListenAndServe(addr, nil)
}

func NewHTTPServer(
	ctx context.Context,
	kubeClients *clients.KubeClients,
	buildInventory inventory.Interface,
) (*HTTPServer, error) {
	buildClientset, err := kubeClients.GetShipwrightClientset()
	if err != nil {
		return nil, err
	}
	clientset, err := kubeClients.GetKubernetesClientset()
	if err != nil {
		return nil, err
	}
	return &HTTPServer{
		ctx:            ctx,
		buildInventory: buildInventory,
		buildClientset: buildClientset,
		clientset:      clientset,
	}, nil
}
