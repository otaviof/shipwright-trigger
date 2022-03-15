package controllers

import (
	"context"
	"log"
	"time"

	"github.com/otaviof/shipwright-trigger/pkg/trigger/clients"
	"github.com/otaviof/shipwright-trigger/pkg/trigger/inventory"
	buildinformers "github.com/shipwright-io/build/pkg/client/informers/externalversions"
	tkninformers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions"
)

// Controller concentrate all other Kubernetes controllers in a single place.
type Controller struct {
	ctx context.Context

	resyncPeriod   time.Duration       // interval to resynchronize all objects
	buildInventory inventory.Interface // build inventory instance

	buildInformerFactory  buildinformers.SharedInformerFactory // shipwright build informer
	tektonInformerFactory tkninformers.SharedInformerFactory   // tekton pipeline informer

	controllersMap map[string]Interface // controller instances indexed by name
}

// Start start informer factory instances, and call "Start" on the controller instances.
func (c *Controller) Start() error {
	log.Print("Starting the Build informer")
	c.buildInformerFactory.Start(c.ctx.Done())
	log.Print("Starting the Tekton informer")
	c.tektonInformerFactory.Start(c.ctx.Done())

	for name, ctrl := range c.controllersMap {
		log.Printf("Starting informers for %q controller...", name)
		if err := ctrl.Start(); err != nil {
			return err
		}
	}
	return nil
}

// Run actionate the event processor on the controller instances ("Run"), capturing all errors on a
// single error channel, as well as holding back waiting for the "Done" on global context.
func (c *Controller) Run() error {
	errorCh := make(chan error)

	for name, ctrl := range c.controllersMap {
		go func(name string, ctrl Interface) {
			log.Printf("Starting event processor for %q controller...", name)
			if err := ctrl.Run(); err != nil {
				log.Fatalf("%s controller error: '%v'", name, err)
				errorCh <- err
			}
		}(name, ctrl)
	}

	// waiting for possible errors on the channel, and holding the method in execution until the
	// "Done" context is issued
	select {
	case err := <-errorCh:
		return err
	case <-c.ctx.Done():
		return nil
	}
}

// bootstrap instantiate all clients, informer factory instances and the actual controllers that will
// be used afterwards.
func (c *Controller) bootstrap(kubeClients *clients.KubeClients) error {
	buildClientset, err := kubeClients.GetShipwrightClientset()
	if err != nil {
		return err
	}
	tektonClientset, err := kubeClients.GetTektonClientset()
	if err != nil {
		return err
	}

	c.buildInformerFactory = buildinformers.NewSharedInformerFactory(buildClientset, c.resyncPeriod)
	c.tektonInformerFactory = tkninformers.NewSharedInformerFactory(tektonClientset, c.resyncPeriod)

	c.controllersMap["shipwright-build"] = NewBuildController(
		c.ctx,
		c.buildInformerFactory.Shipwright().V1alpha1().Builds(),
		c.buildInventory,
	)
	c.controllersMap["tekton-run"] = NewRunController(
		c.ctx,
		c.tektonInformerFactory.Tekton().V1alpha1(),
		tektonClientset,
		c.buildInformerFactory.Shipwright().V1alpha1().BuildRuns(),
		buildClientset,
	)
	c.controllersMap["tekton-pipelinerun"] = NewPipelineRunController(
		c.ctx,
		c.tektonInformerFactory.Tekton().V1beta1(),
		tektonClientset,
		buildClientset,
		c.buildInventory,
	)
	return nil
}

func NewController(
	ctx context.Context,
	kubeClients *clients.KubeClients,
	resyncPeriod time.Duration,
	buildInventory inventory.Interface,
) (*Controller, error) {
	c := &Controller{
		ctx:            ctx,
		resyncPeriod:   resyncPeriod,
		buildInventory: buildInventory,
		controllersMap: map[string]Interface{},
	}
	return c, c.bootstrap(kubeClients)
}
