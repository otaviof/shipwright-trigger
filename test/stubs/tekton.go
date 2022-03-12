package stubs

import (
	"fmt"
	"time"

	tknapisv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var TektonPipelineRunStatusCustomTaskShipwright = &tknapisv1beta1.PipelineSpec{
	Tasks: []tknapisv1beta1.PipelineTask{TektonCustomTaskShipwright},
}

var TektonCustomTaskShipwright = tknapisv1beta1.PipelineTask{
	Name: "shipwright",
	TaskRef: &tknapisv1beta1.TaskRef{
		APIVersion: ShipwrightAPIVersion,
		Name:       "name",
	},
}

func TektonPipelineRunCanceled(name string) tknapisv1beta1.PipelineRun {
	pipelineRun := TektonPipelineRun(name)
	pipelineRun.Spec.Status = tknapisv1beta1.PipelineRunSpecStatus(tknapisv1beta1.PipelineRunReasonCancelled)
	return pipelineRun
}

func TektonPipelineRunRunning(name string) tknapisv1beta1.PipelineRun {
	pipelineRun := TektonPipelineRun(name)
	pipelineRun.Status.StartTime = &metav1.Time{Time: time.Now()}
	return pipelineRun
}

func TektonPipelineRunTimedOut(name string) tknapisv1beta1.PipelineRun {
	pipelineRun := TektonPipelineRun(name)
	pipelineRun.Spec.Timeout = &metav1.Duration{Duration: time.Second}
	pipelineRun.Status.StartTime = &metav1.Time{
		Time: time.Date(2022, time.January, 1, 0, 0, 0, 0, time.Local),
	}
	return pipelineRun
}

func TektonPipelineRunSucceeded(name string) tknapisv1beta1.PipelineRun {
	pipelineRun := TektonPipelineRun(name)
	pipelineRun.Status.MarkSucceeded("Succeeded", fmt.Sprintf("PipelineRun %q has succeeded", name))
	return pipelineRun
}

func TektonPipelineRunFailed(name string) tknapisv1beta1.PipelineRun {
	pipelineRun := TektonPipelineRun(name)
	pipelineRun.Status.MarkFailed("Failed", fmt.Sprintf("PipelineRun %q has failed", name))
	return pipelineRun
}

func TektonPipelineRun(name string) tknapisv1beta1.PipelineRun {
	return tknapisv1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: Namespace,
			Name:      name,
		},
		Spec: tknapisv1beta1.PipelineRunSpec{
			PipelineRef: &tknapisv1beta1.PipelineRef{
				Name: name,
			},
		},
		Status: tknapisv1beta1.PipelineRunStatus{
			PipelineRunStatusFields: tknapisv1beta1.PipelineRunStatusFields{
				PipelineSpec: &tknapisv1beta1.PipelineSpec{},
			},
		},
	}
}
