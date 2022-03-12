package clients

import (
	buildclientset "github.com/shipwright-io/build/pkg/client/clientset/versioned"
	tknclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

type Interface interface {
	GetKubernetesClientset() (kubernetes.Interface, error)
	GetShipwrightClientset() (buildclientset.Interface, error)
	GetTektonClientset() (tknclientset.Interface, error)
}
