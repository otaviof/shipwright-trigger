package webhooks

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/v42/github"
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
)

// GitHubWebHook responsible for handling WebHook requests coming from GitHub, implements Interface.
type GitHubWebHook struct{}

var _ Interface = &GitHubWebHook{}

// ExtractRequestPayload parse the WebHook request in order to read the body payload, and determine
// the type of event based on the headers.
func (g *GitHubWebHook) ExtractRequestPayload(r *http.Request) (*RequestPayload, error) {
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	eventType := github.WebHookType(r)
	if eventType == "" {
		return nil, fmt.Errorf("%w: empty event-type", ErrUnknownEventType)
	}

	return &RequestPayload{
		Payload:   payload,
		EventType: eventType,
		Signature: r.Header.Get(github.SHA256SignatureHeader),
	}, nil
}

// ExtractBuildSelector parses the specific event into its type, and uses specific type attributes
// to construct the BuildSelector instance.
func (g *GitHubWebHook) ExtractBuildSelector(rp *RequestPayload) (*BuildSelector, error) {
	event, err := github.ParseWebHook(rp.EventType, rp.Payload)
	if err != nil {
		return nil, fmt.Errorf("%w: eventType=%q, err=%q", ErrParsingEvent, rp.EventType, err)
	}

	selector := &BuildSelector{}
	switch e := event.(type) {
	case *github.PingEvent:
		log.Printf("Received a Ping event!")
	case *github.PushEvent:
		log.Printf("Received a %q event!", v1alpha1.WhenPush)

		selector.WhenType = v1alpha1.WhenPush

		repo := e.GetRepo()
		if repo == nil {
			return nil, fmt.Errorf("%w: 'repo' is nil", ErrIncompleteEvent)
		}
		selector.RepoURL = e.Repo.GetHTMLURL()
		selector.RepoFullName = e.Repo.GetFullName()

		headCommit := e.GetHeadCommit()
		if headCommit == nil {
			return nil, fmt.Errorf("%w: 'headcommit' is nil", ErrIncompleteEvent)
		}
		selector.Revision = strings.TrimPrefix(*e.Ref, "refs/heads/")
	case *github.PullRequestEvent:
		log.Printf("Received a %q event!", v1alpha1.WhenPullRequest)

		selector.WhenType = v1alpha1.WhenPullRequest
		panic("TODO: PullRequestEvent!!")
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedEventType, e)
	}
	return selector, nil
}

// ValidateSignature validates the informed secret token against the RequestPayload, which contains
// the signature extracted from the initial request.
func (g *GitHubWebHook) ValidateSignature(rp *RequestPayload, secretToken []byte) error {
	return github.ValidateSignature(rp.Signature, rp.Payload, secretToken)
}

// NewGitHubWebHook instantiate GitHub WebHook support.
func NewGitHubWebHook() *GitHubWebHook {
	return &GitHubWebHook{}
}
