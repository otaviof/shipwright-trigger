package controllers

import (
	"reflect"
	"testing"

	buildapisv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	tknapisv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tknapisv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func TestTektonRunParamsToShipwrightParamValues(t *testing.T) {
	value := "value"

	tests := []struct {
		name string
		run  *tknapisv1alpha1.Run
		want []buildapisv1alpha1.ParamValue
	}{{
		name: "run does not contain params",
		run: &tknapisv1alpha1.Run{
			Spec: tknapisv1alpha1.RunSpec{
				Params: []tknapisv1beta1.Param{},
			},
		},
		want: []buildapisv1alpha1.ParamValue{},
	}, {
		name: "run contains an string param",
		run: &tknapisv1alpha1.Run{
			Spec: tknapisv1alpha1.RunSpec{
				Params: []tknapisv1beta1.Param{{
					Name:  "string",
					Value: *tknapisv1beta1.NewArrayOrString(value),
				}},
			},
		},
		want: []buildapisv1alpha1.ParamValue{{
			Name: "string",
			SingleValue: &buildapisv1alpha1.SingleValue{
				Value: &value,
			},
		}},
	}, {
		name: "run contains an string-array param",
		run: &tknapisv1alpha1.Run{
			Spec: tknapisv1alpha1.RunSpec{
				Params: []tknapisv1beta1.Param{{
					Name:  "string-array",
					Value: *tknapisv1beta1.NewArrayOrString(value, value),
				}},
			},
		},
		want: []buildapisv1alpha1.ParamValue{{
			Name: "string-array",
			Values: []buildapisv1alpha1.SingleValue{{
				Value: &value,
			}, {
				Value: &value,
			}},
		}},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TektonRunParamsToShipwrightParamValues(tt.run); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TektonRunParamsToShipwrightParamValues() = %v, want %v", got, tt.want)
			}
		})
	}
}
