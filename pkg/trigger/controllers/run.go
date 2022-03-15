package controllers

import (
	buildapisv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	tknapisv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

// TektonRunParamsToShipwrightParamValues transforms the informed Tekton Run params into Shipwright
// ParamValues slice.
func TektonRunParamsToShipwrightParamValues(
	run *tknapisv1alpha1.Run,
) []buildapisv1alpha1.ParamValue {
	paramValues := []buildapisv1alpha1.ParamValue{}
	for _, p := range run.Spec.Params {
		paramValue := buildapisv1alpha1.ParamValue{Name: p.Name}
		if p.Value.Type == tknapisv1alpha1.ParamTypeArray {
			paramValue.Values = []buildapisv1alpha1.SingleValue{}
			for _, v := range p.Value.ArrayVal {
				paramValue.Values = append(paramValue.Values, buildapisv1alpha1.SingleValue{
					Value: &v,
				})
			}
		} else {
			paramValue.SingleValue = &buildapisv1alpha1.SingleValue{
				Value: &p.Value.StringVal,
			}
		}
		paramValues = append(paramValues, paramValue)
	}
	return paramValues
}
