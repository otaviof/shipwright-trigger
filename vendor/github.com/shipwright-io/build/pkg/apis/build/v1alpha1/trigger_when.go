package v1alpha1

// TriggerWhen a given scenario where the webhook trigger is applicable.
type TriggerWhen struct {
	// Type the event type, the name of the webhook event.
	Type WhenTypeName `json:"type,omitempty"`

	// GitHub describes how to trigger builds based on GitHub (SCM) events.
	//
	// +optional
	GitHub *WhenGitHub `json:"github,omitempty"`

	// Image slice of image names where the event applies.
	//
	// +optional
	Image *WhenImage `json:"image,omitempty"`

	// ObjectRef describes how to match a foreign resource, either by namespace/name or by label
	// selector, plus the given object status.
	//
	// +optional
	ObjectRef *WhenObjectRef `json:"objectRef,omitempty"`
}

// GetBranches return a slice of branch names based on the WhenTypeName informed.
func (w *TriggerWhen) GetBranches(whenType WhenTypeName) []string {
	switch whenType {
	case WhenTypeGitHub:
		if w.GitHub == nil {
			return nil
		}
		return w.GitHub.Branches
	}
	return nil
}
