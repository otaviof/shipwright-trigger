package controllers

import (
	"fmt"
	"log"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	tknapisv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tknapisv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	TektonAPIv1alpha1 = fmt.Sprintf(
		"%s/%s",
		tknapisv1alpha1.SchemeGroupVersion.Group,
		tknapisv1alpha1.SchemeGroupVersion.Version,
	)
	TektonAPIv1beta1 = fmt.Sprintf(
		"%s/%s",
		tknapisv1beta1.SchemeGroupVersion.Group,
		tknapisv1beta1.SchemeGroupVersion.Version,
	)
)

// searchBuildRunForRunOwner inspect the object owners for Tekton Run and returns it, otherwise nil.
func searchBuildRunForRunOwner(br *v1alpha1.BuildRun) *types.NamespacedName {
	for _, ownerRef := range br.OwnerReferences {
		if ownerRef.APIVersion == TektonAPIv1alpha1 && ownerRef.Kind == "Run" {
			return &types.NamespacedName{Namespace: br.GetNamespace(), Name: ownerRef.Name}
		}
	}
	return nil
}

// filterBuildRunOwnedByRun filter out BuildRuns objects not owned by Tekton Run.
func filterBuildRunOwnedByRun(obj interface{}) bool {
	br, ok := obj.(*v1alpha1.BuildRun)
	if !ok {
		return false
	}
	return searchBuildRunForRunOwner(br) != nil
}

// pipelineRunReferencesShipwright checks if the informed PipelineRun is reffering to a Shipwright
// resource via TaskRef.
func pipelineRunReferencesShipwright(pipelineRun *tknapisv1beta1.PipelineRun) bool {
	if pipelineRun.Status.PipelineSpec == nil {
		return false
	}
	for _, task := range pipelineRun.Status.PipelineSpec.Tasks {
		if task.TaskRef == nil {
			continue
		}
		if task.TaskRef.APIVersion == ShipwrightAPIVersion {
			return true
		}
	}
	return false
}

// pipelineRunNameMatchesLabel check if the label added to mark PipelineRun instances is meant for
// the current instance.
func pipelineRunNameMatchesLabel(pipelineRun *tknapisv1beta1.PipelineRun) bool {
	labels := pipelineRun.GetLabels()
	if labels == nil {
		return false
	}
	name, exists := labels[PipelineRunNameKey]
	return exists && pipelineRun.GetName() == name
}

// pipelineRunNotSyncedAndNotCustomTask filters out the PipelineRuns that have already synced, and
// also filter out the instances issued for a Custom-Task. When the instance is synced it will
// receive a label with its name, and so we can detect PipelineRun re-runs.
func pipelineRunNotSyncedAndNotCustomTask(obj interface{}) bool {
	pipelineRun, ok := obj.(*tknapisv1beta1.PipelineRun)
	if !ok {
		log.Printf("Unable to cast object as Tekton PipelineRun: '%#v'", obj)
		return false
	}

	// when the PipelineSpec is nil the PipelineRun is not ready for execution, so we can filter out
	// those instances and wait for the updates
	if pipelineRun.Status.PipelineSpec == nil {
		return false
	}
	// making sure the instance is not part of a shipwright custom-task
	if pipelineRunReferencesShipwright(pipelineRun) {
		return false
	}
	// checks the label to assert if it was already synced
	return !pipelineRunNameMatchesLabel(pipelineRun)
}
