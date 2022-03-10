// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	// NamespacedBuildStrategyKind indicates that the buildstrategy type has a namespaced scope.
	NamespacedBuildStrategyKind BuildStrategyKind = "BuildStrategy"

	// ClusterBuildStrategyKind indicates that buildstrategy type has a cluster scope.
	ClusterBuildStrategyKind BuildStrategyKind = "ClusterBuildStrategy"
)

// BuildStrategySpec defines the desired state of BuildStrategy
type BuildStrategySpec struct {
	BuildSteps []BuildStep `json:"buildSteps,omitempty"`
	Parameters []Parameter `json:"parameters,omitempty"`
}

// ParameterType indicates the type of a parameter
type ParameterType string

// Valid ParamTypes:
const (
	ParameterTypeString ParameterType = "string"
	ParameterTypeArray  ParameterType = "array"
)

// Parameter holds a name-description with a default value
// that allows strategy steps to be parameterize.
// Build users can set a value for parameter via the Build
// or BuildRun spec.paramValues object.
// +optional
type Parameter struct {
	// Name of the parameter
	// +required
	Name string `json:"name"`

	// Description on the parameter purpose
	// +required
	Description string `json:"description"`

	// Type of the parameter. The possible types are "string" and "array",
	// and "string" is the default.
	// +optional
	Type ParameterType `json:"type,omitempty"`

	// Default value for a string parameter
	// +optional
	Default *string `json:"default,omitempty"`

	// Default values for an array parameter
	// +optional
	Defaults *[]string `json:"defaults"`
}

// BuildStep defines a partial step that needs to run in container for building the image.
// If the build step declares a volumeMount, Shipwright will create an emptyDir volume mount for the named volume.
// Build steps which share the same named volume in the volumeMount will share the same underlying emptyDir volume.
// This behavior is deprecated, and will be removed when full volume support is added to build strategies as specified
// in SHIP-0022.
type BuildStep struct {
	corev1.Container `json:",inline"`
}

// BuildStrategyStatus defines the observed state of BuildStrategy
type BuildStrategyStatus struct {
}

// BuildStrategyKind defines the type of BuildStrategy used by the build.
type BuildStrategyKind string

// Strategy can be used to refer to a specific instance of a buildstrategy.
// Copied from CrossVersionObjectReference: https://github.com/kubernetes/kubernetes/blob/169df7434155cbbc22f1532cba8e0a9588e29ad8/pkg/apis/autoscaling/types.go#L64
type Strategy struct {
	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name"`

	// BuildStrategyKind indicates the kind of the buildstrategy, namespaced or cluster scoped.
	Kind *BuildStrategyKind `json:"kind,omitempty"`

	// API version of the referent
	// +optional
	APIVersion *string `json:"apiVersion,omitempty"`
}

// BuilderStrategy defines the common elements of build strategies
type BuilderStrategy interface {
	GetAnnotations() map[string]string
	GetName() string
	GetGeneration() int64
	GetResourceLabels() map[string]string
	GetBuildSteps() []BuildStep
	GetParameters() []Parameter
}
