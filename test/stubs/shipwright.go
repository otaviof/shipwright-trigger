package stubs

import (
	"fmt"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildapisv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const Namespace = "namespace"

var ShipwrightAPIVersion = fmt.Sprintf(
	"%s/%s",
	buildapisv1alpha1.SchemeGroupVersion.Group,
	buildapisv1alpha1.SchemeGroupVersion.Version,
)

var TriggerWhenPushToMain = v1alpha1.TriggerWhen{
	Type:     v1alpha1.WhenPush,
	Branches: []string{"main"},
}

var TriggerWhenPipelineSucceeded = v1alpha1.TriggerWhen{
	Type: v1alpha1.WhenPipeline,
	ObjectRef: &v1alpha1.ObjectRef{
		Name:   "pipeline",
		Status: []string{"Succeeded"},
	},
}

func ShipwrightBuild(name string) v1alpha1.Build {
	return v1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: Namespace,
			Name:      name,
		},
		Spec: v1alpha1.BuildSpec{
			Source: v1alpha1.Source{
				URL: &RepoURL,
			},
		},
	}
}

func ShipwrightBuildWithTriggers(name string, triggers ...v1alpha1.TriggerWhen) v1alpha1.Build {
	b := ShipwrightBuild(name)
	b.Spec.Trigger = &v1alpha1.Trigger{When: triggers}
	return b
}
