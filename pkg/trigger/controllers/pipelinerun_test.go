package controllers

import (
	"testing"

	"github.com/otaviof/shipwright-trigger/test/stubs"
	tknapisv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func TestParsePipelineRunStatus(t *testing.T) {
	tests := []struct {
		name        string
		pipelineRun tknapisv1beta1.PipelineRun
		want        string
		wantErr     bool
	}{{
		name:        "cancelled",
		pipelineRun: stubs.TektonPipelineRunCanceled("name"),
		want:        "Cancelled",
		wantErr:     false,
	}, {
		name:        "started",
		pipelineRun: stubs.TektonPipelineRunRunning("name"),
		want:        "Started",
		wantErr:     false,
	}, {
		name:        "timedout",
		pipelineRun: stubs.TektonPipelineRunTimedOut("name"),
		want:        "TimedOut",
		wantErr:     false,
	}, {
		name:        "succeeded",
		pipelineRun: stubs.TektonPipelineRunSucceeded("name"),
		want:        "Succeeded",
		wantErr:     false,
	}, {
		name:        "failed",
		pipelineRun: stubs.TektonPipelineRunFailed("name"),
		want:        "Failed",
		wantErr:     false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePipelineRunStatus(&tt.pipelineRun)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePipelineRunStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParsePipelineRunStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}
