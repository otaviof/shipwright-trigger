package controllers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/otaviof/shipwright-trigger/pkg/trigger/inventory"
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildclientset "github.com/shipwright-io/build/pkg/client/clientset/versioned"
	tknapisv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tknclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tkninformerv1beta1 "github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1beta1"
	tknlisterv1beta1 "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// PipelineRunController watches for PipelineRun objects checking the Shipwright Builds that will
// need to be triggered.
type PipelineRunController struct {
	ctx context.Context

	informer       tkninformerv1beta1.Interface       // tekton pipelinerun informer
	lister         tknlisterv1beta1.PipelineRunLister // tekton pipelinerun lister
	informerSynced cache.InformerSynced               // informer synced status
	clientset      tknclientset.Interface             // tekton clientset
	buildClientset buildclientset.Interface           // shipwright build clientset
	wq             workqueue.RateLimitingInterface    // controller workqueue

	buildInventory inventory.Interface // build triggers inventory
}

var _ Interface = &PipelineRunController{}

// createBuildRun handles the actual BuildRun creation, uses the informed PipelineRun instance to
// establish ownership. Only returns the created object name, and error.
func (c *PipelineRunController) createBuildRun(
	pipelineRun *tknapisv1beta1.PipelineRun,
	buildName string,
) (string, error) {
	buildClient := c.buildClientset.ShipwrightV1alpha1().BuildRuns(pipelineRun.GetNamespace())
	br, err := buildClient.Create(c.ctx, &v1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", pipelineRun.GetName()),
			Labels: map[string]string{
				OwnedByPipelineRunLabelKey: pipelineRun.GetName(),
			},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: TektonAPIv1beta1,
				Kind:       "PipelineRun",
				Name:       pipelineRun.GetName(),
				UID:        pipelineRun.GetUID(),
			}},
		},
		Spec: v1alpha1.BuildRunSpec{
			BuildRef: v1alpha1.BuildRef{
				Name: buildName,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	return br.GetName(), nil
}

// triggerBuildsForPipelineRun create the BuildRun instances for the informed objects, and updates
// the PipelineRun object labels to list the created objects.
func (c *PipelineRunController) triggerBuildsForPipelineRun(
	pipelineRun *tknapisv1beta1.PipelineRun,
	buildsToBeTriggered []inventory.SearchResult,
) error {
	var created []string
	for _, build := range buildsToBeTriggered {
		buildRunName, err := c.createBuildRun(pipelineRun, build.BuildName.Name)
		if err != nil {
			return err
		}
		created = append(created, buildRunName)
	}
	if len(created) == 0 {
		return fmt.Errorf("no buildruns have been created for %q", pipelineRun.GetNamespacedName())
	}
	log.Printf("BuildRun(s) %q have been created for %q", created, pipelineRun.GetNamespacedName())

	// adding a label to the PipelineRun object to identify the BuildRun(s) created for it, and also,
	// to be able to filter out objects that have already triggered Builds afterwards
	labels := pipelineRun.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[BuildRunsCreatedKey] = strings.Join(created, ", ")
	labels[PipelineRunNameKey] = pipelineRun.GetName()
	pipelineRun.SetLabels(labels)
	_, err := c.clientset.TektonV1beta1().
		PipelineRuns(pipelineRun.GetNamespace()).
		Update(c.ctx, pipelineRun, metav1.UpdateOptions{})
	return err
}

// sync inspect PipelineRun to extract the query parameters for the Build inventory search.
func (c *PipelineRunController) sync(key string) error {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	log.Printf("Syncing Tekton PipelineRun named '%s/%s'...", ns, name)

	pipelineRun, err := c.lister.PipelineRuns(ns).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if pipelineRun.Spec.PipelineRef == nil {
		log.Printf("PipelineRun does not point to a Pipeline, skipping!")
		return nil
	}

	// if the label recording the original PipelineRun name, which triggered builds, matches the
	// current object name it gets skipped from the rest of the syncing process
	if pipelineRunNameMatchesLabel(pipelineRun) {
		log.Print("PipelineRun already triggered Shipwright Build(s)")
		return nil
	}

	// creating a objectRef based on the informed PipelineRun, the instance is informed to the
	// inventory query interface to list Shipwright Builds that should be triggered
	objectRef, err := PipelineRunToObjectRef(pipelineRun)
	if err != nil {
		return err
	}
	log.Printf("Searching for Builds matching: name=%q, status=%q, matchLabels=%q",
		objectRef.Name, objectRef.Status, objectRef.Selector)
	buildsToBeTriggered := c.buildInventory.SearchForObjectRef(v1alpha1.WhenPipeline, objectRef)
	if len(buildsToBeTriggered) == 0 {
		return nil
	}
	return c.triggerBuildsForPipelineRun(pipelineRun, buildsToBeTriggered)
}

func (c *PipelineRunController) processor() {
	for processNextItem(c.wq, c.sync) {
	}
}

// Start wait for the informer cache synchronization.
func (c *PipelineRunController) Start() error {
	log.Printf("Waiting for Tekton PipelineRun informer cache synchronization")
	if !cache.WaitForCacheSync(c.ctx.Done(), c.informerSynced) {
		return fmt.Errorf("tekton pipelinerun informer is not synced")
	}
	return nil
}

// Run activate event processor until the context is done.
func (c *PipelineRunController) Run() error {
	defer c.wq.ShutDown()

	log.Printf("Starting Tekton PipelineRun event processor")
	go wait.Until(c.processor, 100*time.Millisecond, c.ctx.Done())

	log.Printf("Tekton PipelineRun controller is running!")
	<-c.ctx.Done()
	log.Printf("Tekton PipelineRun controller is shutting down..")
	return nil
}

// NewPipelineRunController instantiate the controller.
func NewPipelineRunController(
	ctx context.Context,
	informer tkninformerv1beta1.Interface,
	clientset tknclientset.Interface,
	buildClientset buildclientset.Interface,
	buildInventory inventory.Interface,
) *PipelineRunController {
	wq := workqueue.NewNamedRateLimitingQueue(
		workqueue.DefaultControllerRateLimiter(),
		"pipelineruns",
	)
	c := &PipelineRunController{
		ctx: ctx,

		informer:       informer,
		lister:         informer.PipelineRuns().Lister(),
		informerSynced: informer.PipelineRuns().Informer().HasSynced,
		clientset:      clientset,
		buildClientset: buildClientset,
		wq:             wq,

		buildInventory: buildInventory,
	}
	// the PipelineRun objects that have already triggered BuildRuns are filtered out, all other
	// objects are enqueued and synced regularly
	informer.PipelineRuns().Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: pipelineRunNotSyncedAndNotCustomTask,
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    enqueuePipelineRunFn(wq),
			UpdateFunc: compareAndEnqueuePipelineRunFn(wq),
			DeleteFunc: enqueuePipelineRunFn(wq),
		},
	})
	return c
}
