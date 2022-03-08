package controllers

import (
	"reflect"
	"testing"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	tknapisv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func Test_searchBuildRunForRunOwner(t *testing.T) {
	tests := []struct {
		name string
		br   *v1alpha1.BuildRun
		want *types.NamespacedName
	}{{
		name: "buildrun not owned by tekton run",
		br: &v1alpha1.BuildRun{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{},
			},
		},
		want: nil,
	}, {
		name: "buildrun owned by tekton run",
		br: &v1alpha1.BuildRun{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{{
					APIVersion: TektonAPIv1alpha1,
					Kind:       "Run",
					Name:       "run",
				}},
				Namespace: "namespace",
				Name:      "buildrun",
			},
		},
		want: &types.NamespacedName{Namespace: "namespace", Name: "run"},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := searchBuildRunForRunOwner(tt.br); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("searchBuildRunForRunOwner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pipelineRunReferencesShipwright(t *testing.T) {
	tests := []struct {
		name        string
		pipelineRun *tknapisv1beta1.PipelineRun
		want        bool
	}{{
		name: "pipelinerun has status.pipelinespec nil",
		pipelineRun: &tknapisv1beta1.PipelineRun{
			Status: tknapisv1beta1.PipelineRunStatus{
				PipelineRunStatusFields: tknapisv1beta1.PipelineRunStatusFields{
					PipelineSpec: nil,
				},
			},
		},
		want: false,
	}, {
		name: "pipelinerun does not references shipwright build",
		pipelineRun: &tknapisv1beta1.PipelineRun{
			Status: tknapisv1beta1.PipelineRunStatus{
				PipelineRunStatusFields: tknapisv1beta1.PipelineRunStatusFields{
					PipelineSpec: &tknapisv1beta1.PipelineSpec{
						Tasks: []tknapisv1beta1.PipelineTask{{}},
					},
				},
			},
		},
		want: false,
	}, {
		name: "pipelinerun references shipwright build",
		pipelineRun: &tknapisv1beta1.PipelineRun{
			Status: tknapisv1beta1.PipelineRunStatus{
				PipelineRunStatusFields: tknapisv1beta1.PipelineRunStatusFields{
					PipelineSpec: &tknapisv1beta1.PipelineSpec{
						Tasks: []tknapisv1beta1.PipelineTask{{
							Name: "task",
							TaskRef: &tknapisv1beta1.TaskRef{
								Name:       "shipwright-build",
								APIVersion: ShipwrightAPIVersion,
								Kind:       "Build",
							},
						}},
					},
				},
			},
		},
		want: true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pipelineRunReferencesShipwright(tt.pipelineRun); got != tt.want {
				t.Errorf("pipelineRunReferencesShipwright() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pipelineRunNameMatchesLabel(t *testing.T) {
	tests := []struct {
		name        string
		pipelineRun *tknapisv1beta1.PipelineRun
		want        bool
	}{{
		name: "empty instance",
		pipelineRun: &tknapisv1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace",
				Name:      "name",
			},
		},
		want: false,
	}, {
		name: "name label does not match current name",
		pipelineRun: &tknapisv1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					PipelineRunNameKey: "another-name",
				},
				Namespace: "namespace",
				Name:      "name",
			},
		},
		want: false,
	}, {
		name: "name label  matches current name",
		pipelineRun: &tknapisv1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					PipelineRunNameKey: "name",
				},
				Namespace: "namespace",
				Name:      "name",
			},
		},
		want: true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pipelineRunNameMatchesLabel(tt.pipelineRun); got != tt.want {
				t.Errorf("pipelineRunNameMatchesLabel() = %v, want %v", got, tt.want)
			}
		})
	}
}
