package v1alpha1

// WhenTypeName set of TriggerWhen valid names.
type WhenTypeName string

const (
	// WhenTypeGitHub GitHub trigger type name.
	WhenTypeGitHub WhenTypeName = "GitHub"

	// WhenTypeImage Image trigger type name.
	WhenTypeImage WhenTypeName = "Image"

	// WhenTypePipeline Tekton Pipeline trigger type name.
	WhenTypePipeline WhenTypeName = "Pipeline"
)

// GitHubEventName set of WhenGitHub valid event names.
type GitHubEventName string

const (
	// GitHubPullRequestEvent github pull-request event name.
	GitHubPullRequestEvent GitHubEventName = "PullRequest"

	// GitHubPushEvent git push webhook event name.
	GitHubPushEvent GitHubEventName = "Push"
)

// WhenImage attributes to match Image events.
type WhenImage struct {
	// Names fully qualified image names.
	//
	// +optional
	Names []string `json:"names,omitempty"`
}

// WhenGitHub attributes to match GitHub events.
type WhenGitHub struct {
	// Events GitHub event names.
	//
	// +optional
	Events []GitHubEventName `json:"events,omitempty"`

	// Branches slice of branch names where the event applies.
	//
	// +optional
	Branches []string `json:"branches,omitempty"`
}

// WhenObjectRef attributes to reference local Kubernetes objects.
type WhenObjectRef struct {
	// Name target object name.
	//
	// +optional
	Name string `json:"name,omitempty"`

	// Status object status.
	Status []string `json:"status,omitempty"`

	// Selector label selector.
	//
	// +optional
	Selector map[string]string `json:"selector,omitempty"`
}
