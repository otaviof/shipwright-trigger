package clients

import (
	buildclientset "github.com/shipwright-io/build/pkg/client/clientset/versioned"
	fakebuildclientset "github.com/shipwright-io/build/pkg/client/clientset/versioned/fake"
	tknclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	faketknclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type FakeKubeClients struct {
	objects []runtime.Object
}

var _ Interface = &FakeKubeClients{}

func (c *FakeKubeClients) GetKubernetesClientset() (kubernetes.Interface, error) {
	return fake.NewSimpleClientset(c.objects...), nil
}

func (c *FakeKubeClients) GetShipwrightClientset() (buildclientset.Interface, error) {
	return fakebuildclientset.NewSimpleClientset(c.objects...), nil
}

func (c *FakeKubeClients) GetTektonClientset() (tknclientset.Interface, error) {
	return faketknclientset.NewSimpleClientset(c.objects...), nil
}

func NewFakeKubeClients(objects ...runtime.Object) *FakeKubeClients {
	return &FakeKubeClients{
		objects: objects,
	}
}
