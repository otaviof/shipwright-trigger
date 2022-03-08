package webhooks

import "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"

// BuildSelector defines the group of attributes to select the respective Build instance.
type BuildSelector struct {
	WhenType     v1alpha1.WhenType // webhook trigger type
	RepoURL      string            // repository URL
	RepoFullName string            // repository full name
	Revision     string            // repository revision
}

// IsEmpty checks if RepoURL is empty.
func (b *BuildSelector) IsEmpty() bool {
	return b.RepoURL == ""
}
