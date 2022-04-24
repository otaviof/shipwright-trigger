package stubs

import (
	"fmt"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const Namespace = "namespace"

var ShipwrightAPIVersion = fmt.Sprintf(
	"%s/%s",
	v1alpha1.SchemeGroupVersion.Group,
	v1alpha1.SchemeGroupVersion.Version,
)

var TriggerWhenPushToMain = v1alpha1.TriggerWhen{
	Type: v1alpha1.WhenTypeGitHub,
	GitHub: &v1alpha1.WhenGitHub{
		Events: []v1alpha1.GitHubEventName{
			v1alpha1.GitHubPushEvent,
		},
		Branches: []string{"main"},
	},
}

var TriggerWhenPipelineSucceeded = v1alpha1.TriggerWhen{
	Type: v1alpha1.WhenTypePipeline,
	ObjectRef: &v1alpha1.WhenObjectRef{
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
