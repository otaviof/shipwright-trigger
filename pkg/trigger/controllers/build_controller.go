package controllers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/otaviof/shipwright-trigger/pkg/trigger/inventory"
	buildinformer "github.com/shipwright-io/build/pkg/client/informers/externalversions/build/v1alpha1"
	buildlister "github.com/shipwright-io/build/pkg/client/listers/build/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// BuildController is a Kubernetes BuildController watching over Build objects and storing them on
// the Inventory instance.
type BuildController struct {
	ctx context.Context

	informer       buildinformer.BuildInformer     // build informer instance
	lister         buildlister.BuildLister         // informer's lister
	informerSynced cache.InformerSynced            // informer synced status
	wq             workqueue.RateLimitingInterface // workqueue instance

	buildInventory inventory.Interface // inventory instance
}

var _ Interface = &BuildController{}

// sync handle the synchronization of the resources with the Inventory.
func (c *BuildController) sync(key string) error {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	b, err := c.lister.Builds(ns).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			c.buildInventory.Remove(types.NamespacedName{Namespace: ns, Name: name})
			return nil
		}
		return err
	}

	c.buildInventory.Add(b)
	return nil
}

func (c *BuildController) processor() {
	for processNextItem(c.wq, c.sync) {
	}
}

// Start waits for the informer cache to synchronize.
func (c *BuildController) Start() error {
	log.Printf("Waiting for Shipwright Build informer cache synchronization")
	if !cache.WaitForCacheSync(c.ctx.Done(), c.informerSynced) {
		return fmt.Errorf("build informer is not synced")
	}
	return nil
}

// Run the event processor loop.
func (c *BuildController) Run() error {
	defer c.wq.ShutDown()

	log.Printf("Starting Shipwright Build event processor")
	go wait.Until(c.processor, 100*time.Millisecond, c.ctx.Done())

	log.Printf("Shipwright Build controller is running!")
	<-c.ctx.Done()
	log.Printf("Shipwright Build controller is shutting down...")
	return nil
}

// NewBuildController instantiate the Controller by putting together the informer, lister and event
// handler for Build objects.
func NewBuildController(
	ctx context.Context,
	informer buildinformer.BuildInformer,
	buildInventory inventory.Interface,
) *BuildController {
	wq := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "builds")
	c := &BuildController{
		ctx: ctx,

		informer:       informer,
		lister:         informer.Lister(),
		informerSynced: informer.Informer().HasSynced,
		wq:             wq,

		buildInventory: buildInventory,
	}
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    enqueueBuildFn(wq),
		UpdateFunc: compareAndEnqueueBuildFn(wq),
		DeleteFunc: enqueueBuildFn(wq),
	})
	return c
}
