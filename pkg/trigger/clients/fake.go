package clients

import (
	"fmt"

	buildclientset "github.com/shipwright-io/build/pkg/client/clientset/versioned"
	fakebuildclientset "github.com/shipwright-io/build/pkg/client/clientset/versioned/fake"
	tknclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	faketknclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// FakeKubeClients implements the Interface for testing with fake clientsets.
type FakeKubeClients struct {
	objects         []runtime.Object              // list of objects
	clientset       *fake.Clientset               // kubernetes clientset
	buildClientset  *fakebuildclientset.Clientset // shipwright clientset
	tektonClientset *faketknclientset.Clientset   // tekton clientset
}

var _ Interface = &FakeKubeClients{}

// generateNameReactor intercepts object creation action in order to simulate the "generateName" in
// Kubernetes, where a object created with the directive receives a random name suffix.
func generateNameReactor(action k8stesting.Action) (bool, runtime.Object, error) {
	createAction, ok := action.(k8stesting.CreateAction)
	if !ok {
		panic(fmt.Errorf("action is not a create action: %+v", action))
	}

	obj := createAction.GetObject()
	objMeta, err := meta.Accessor(obj)
	if err != nil {
		panic(err)
	}

	if objMeta.GetName() == "" {
		genName := objMeta.GetGenerateName()
		if genName == "" {
			panic(fmt.Errorf("object does not have a name or generateName: '%#v'", obj))
		}
		suffix := rand.String(5)
		objMeta.SetName(fmt.Sprintf("%s%s", genName, suffix))
	}
	return false, nil, nil
}

// GetKubernetesClientset returns or instantiate a new fake Kubernetes clientset.
func (c *FakeKubeClients) GetKubernetesClientset() (kubernetes.Interface, error) {
	if c.clientset != nil {
		return c.clientset, nil
	}

	c.clientset = fake.NewSimpleClientset(c.objects...)
	c.clientset.PrependReactor("create", "*", generateNameReactor)
	return c.clientset, nil
}

// GetShipwrightClientset returns or instantiate a new fake Shipwright clientset.
func (c *FakeKubeClients) GetShipwrightClientset() (buildclientset.Interface, error) {
	if c.buildClientset != nil {
		return c.buildClientset, nil
	}

	c.buildClientset = fakebuildclientset.NewSimpleClientset(c.objects...)
	c.buildClientset.PrependReactor("create", "*", generateNameReactor)
	return c.buildClientset, nil
}

// GetTektonClientset returns or instantiate a new fake Tekton clientset.
func (c *FakeKubeClients) GetTektonClientset() (tknclientset.Interface, error) {
	if c.tektonClientset != nil {
		return c.tektonClientset, nil
	}

	c.tektonClientset = faketknclientset.NewSimpleClientset(c.objects...)
	c.tektonClientset.PrependReactor("create", "*", generateNameReactor)
	return c.tektonClientset, nil
}

// NewFakeKubeClients instantiate the FakeKubeClients with the objects every client will receive.
func NewFakeKubeClients(objects ...runtime.Object) *FakeKubeClients {
	return &FakeKubeClients{
		objects: objects,
	}
}
