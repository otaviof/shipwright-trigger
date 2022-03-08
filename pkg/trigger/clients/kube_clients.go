package clients

import (
	buildclientset "github.com/shipwright-io/build/pkg/client/clientset/versioned"
	tknclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// KubeClients manaages all clients needed to interact with Kubernetes in a single place.
type KubeClients struct {
	flags *genericclioptions.ConfigFlags // command-line flags

	restConfig      *rest.Config              // base rest config instance
	clientset       *kubernetes.Clientset     // kubernetes clientset
	buildClientset  *buildclientset.Clientset // shipwright build clientset
	tektonClientset *tknclientset.Clientset   // tekton pipelines clientset
}

// GetRestConfig returns the existing rest.Config instance, or instantiate one.
func (c *KubeClients) GetRestConfig() (*rest.Config, error) {
	if c.restConfig != nil {
		return c.restConfig, nil
	}

	clientConfigLoader := c.flags.ToRawKubeConfigLoader()

	var err error
	if c.restConfig, err = clientConfigLoader.ClientConfig(); err != nil {
		return nil, err
	}
	return c.restConfig, nil
}

// GetKubernetesClientset returns the existing Kubernetes clientset instance, or instantiate one.
func (c *KubeClients) GetKubernetesClientset() (*kubernetes.Clientset, error) {
	if c.clientset != nil {
		return c.clientset, nil
	}

	var err error
	if c.clientset, err = kubernetes.NewForConfig(c.restConfig); err != nil {
		return nil, err
	}
	return c.clientset, nil
}

// GetShipwrightClientset returns the existing Shipwright clientset instance, or instantiate one.
func (c *KubeClients) GetShipwrightClientset() (*buildclientset.Clientset, error) {
	if c.buildClientset != nil {
		return c.buildClientset, nil
	}

	var err error
	if c.buildClientset, err = buildclientset.NewForConfig(c.restConfig); err != nil {
		return nil, err
	}
	return c.buildClientset, nil
}

// GetTektonClientset returns the existing Tekton Pipelines clientset instance, or instantiate one.
func (c *KubeClients) GetTektonClientset() (*tknclientset.Clientset, error) {
	if c.tektonClientset != nil {
		return c.tektonClientset, nil
	}

	var err error
	if c.tektonClientset, err = tknclientset.NewForConfig(c.restConfig); err != nil {
		return nil, err
	}
	return c.tektonClientset, nil
}

// NewKubeClients intantiate a new KubeClients.
func NewKubeClients(flags *genericclioptions.ConfigFlags) (*KubeClients, error) {
	kubeClients := &KubeClients{flags: flags}
	if _, err := kubeClients.GetRestConfig(); err != nil {
		return nil, err
	}
	return kubeClients, nil
}
