// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

// WhenType represents the webhook event type.
type WhenType string

const (
	// WhenPullRequest github pull-request webhook event.
	WhenPullRequest WhenType = "PullRequest"

	// WhenPush git push webhook event .
	WhenPush WhenType = "Push"

	// WhenImage image change event.
	WhenImage WhenType = "Image"

	// WhenPipeline tekton pipeline event.
	WhenPipeline WhenType = "Pipeline"
)

// ObjectRef references a local object.
type ObjectRef struct {
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

// TriggerWhen a given scenario where the webhook trigger is applicable.
type TriggerWhen struct {
	// Type the event type, the name of the webhook event.
	Type WhenType `json:"type,omitempty"`

	// Branches slice of branch names where the event applies.
	//
	// +optional
	Branches []string `json:"branches,omitempty"`

	// Images slice of image names where the event applies.
	//
	// +optional
	Images []string `json:"images,omitempty"`

	// ObjectRef describes how to match a foreign resource, either by namespace/name or by label
	// selector, plus the given object status.
	//
	// +optional
	ObjectRef *ObjectRef `json:"objectRef,omitempty"`
}

// Trigger represents the webhook trigger configuration for a Build.
type Trigger struct {
	// When the list of scenarios when a new build should take place.
	When []TriggerWhen `json:"when,omitempty"`

	// SecretRef points to a local object carrying the secret token to validate webhook request.
	//
	// +optional
	SecretRef *corev1.LocalObjectReference `json:"secretRef,omitempty"`
}
