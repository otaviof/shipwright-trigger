package controllers

import (
	"fmt"
	"strings"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	tknapisv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/apis"
)

// LabelKeyPrefix prefix used in all labels.
const LabelKeyPrefix = "trigger.shipwright.io"

var (
	// OwnedByRunLabelKey labels the BuildRun as owned by Tekton Run.
	OwnedByRunLabelKey = fmt.Sprintf("%s/owned-by-run", LabelKeyPrefix)
	// OwnedByPipelineRunLabelKey lables the BuildRun as owned by Tekton PipelineRun.
	OwnedByPipelineRunLabelKey = fmt.Sprintf("%s/owned-by-pipelinerun", LabelKeyPrefix)
	// BuildRunsCreatedKey labels the PipelineRun with the BuildRuns created.
	BuildRunsCreatedKey = fmt.Sprintf("%s/buildrun-names", LabelKeyPrefix)
	// PipelineRunNameKey labels PipelineRuns with its current name, to avoid object reprocessing.
	PipelineRunNameKey = fmt.Sprintf("%s/pipelinerun-name", LabelKeyPrefix)
)

// ParsePipelineRunStatus parte the informed object status to extract its status.
func ParsePipelineRunStatus(pipelineRun *tknapisv1beta1.PipelineRun) (string, error) {
	switch {
	case pipelineRun.IsDone():
		if pipelineRun.Status.GetCondition(apis.ConditionSucceeded).IsTrue() {
			return tknapisv1beta1.PipelineRunReasonSuccessful.String(), nil
		}
		return tknapisv1beta1.PipelineRunReasonFailed.String(), nil
	case pipelineRun.IsCancelled():
		return tknapisv1beta1.PipelineRunReasonCancelled.String(), nil
	case pipelineRun.IsTimedOut():
		return tknapisv1beta1.PipelineRunReasonTimedOut.String(), nil
	case pipelineRun.HasStarted():
		return tknapisv1beta1.PipelineRunReasonStarted.String(), nil
	default:
		return "", fmt.Errorf("unable to parse pipelinerun %s current status",
			pipelineRun.GetNamespacedName())
	}
}

// PipelineRunToObjectRef transforms the informed PipelineRun instance to a ObjectRef.
func PipelineRunToObjectRef(pipelineRun *tknapisv1beta1.PipelineRun) (*v1alpha1.ObjectRef, error) {
	status, err := ParsePipelineRunStatus(pipelineRun)
	if err != nil {
		return nil, err
	}
	// cleaning up the labels set by the itself
	labels := pipelineRun.GetLabels()
	for k := range labels {
		if strings.HasPrefix(k, LabelKeyPrefix) {
			delete(labels, k)
		}
	}
	return &v1alpha1.ObjectRef{
		Name:     pipelineRun.Spec.PipelineRef.Name,
		Status:   []string{status},
		Selector: labels,
	}, nil
}
