package controllers

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildapisv1alpha1 "github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildclientset "github.com/shipwright-io/build/pkg/client/clientset/versioned"
	buildinformer "github.com/shipwright-io/build/pkg/client/informers/externalversions/build/v1alpha1"
	buildlister "github.com/shipwright-io/build/pkg/client/listers/build/v1alpha1"
	tknapisv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tknclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tkninformerv1alpha1 "github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1alpha1"
	tknlisterv1alpha1 "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1alpha1"
	tkncontroller "github.com/tektoncd/pipeline/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"knative.dev/pkg/apis"
	knativev1 "knative.dev/pkg/apis/duck/v1"
)

// RunController watches for Tekton Run objects having .ref pointing to a Shipwright Build, when it
// happens this RunController extracts the Build object name to issue a BuildRun. The RunController
// also keeps watching the BuildRun instances owned by Tekton Run resources, in order to follow up
// the BuildRun status updates, reflecting those on the Tekton Run parent. By making sure the status
// is kept up to date, Tekton Pipelines can identify when the Tekton Run is done.
type RunController struct {
	m   sync.Mutex
	ctx context.Context

	runInformer       tkninformerv1alpha1.Interface // run informer
	runLister         tknlisterv1alpha1.RunLister   // run lister
	runInformerSynced cache.InformerSynced          // run informer synced function
	tektonClientset   tknclientset.Interface        // tekton clientset

	buildRunInformer       buildinformer.BuildRunInformer // buildrun informer
	buildRunLister         buildlister.BuildRunLister     // buildrun lister
	buildRunInformerSynced cache.InformerSynced           // buildrun informer synced function
	buildClientset         buildclientset.Interface       // shipwright clientset

	wq workqueue.RateLimitingInterface // controller's workqueue
}

var ShipwrightAPIVersion = fmt.Sprintf(
	"%s/%s",
	buildapisv1alpha1.SchemeGroupVersion.Group,
	buildapisv1alpha1.SchemeGroupVersion.Version,
)

// createBuildRun creates a new BuildRun instance using the informed Tekton Run to establish the
// ownership and to identify the Build resource name. If there are Run parameters, those are copied
// over to the new Buildrun.
func (c *RunController) createBuildRun(run *tknapisv1alpha1.Run) (*v1alpha1.BuildRun, error) {
	buildClient := c.buildClientset.ShipwrightV1alpha1().BuildRuns(run.GetNamespace())
	return buildClient.Create(c.ctx, &v1alpha1.BuildRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", run.Name),
			Labels: map[string]string{
				OwnedByRunLabelKey: run.Name,
			},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: TektonAPIv1alpha1,
				Kind:       "Run",
				Name:       run.GetName(),
				UID:        run.GetUID(),
			}},
		},
		Spec: v1alpha1.BuildRunSpec{
			ParamValues: TektonRunParamsToShipwrightParamValues(run),
			BuildRef: v1alpha1.BuildRef{
				APIVersion: &run.Spec.Ref.APIVersion,
				Name:       run.Spec.Ref.Name,
			},
		},
	}, metav1.CreateOptions{})
}

// updateRunStatus reflect the BuildRun status into the Tekton Run resource.
func (c *RunController) updateRunStatus(run *tknapisv1alpha1.Run, br *v1alpha1.BuildRun) error {
	c.m.Lock()
	defer c.m.Unlock()

	run.Status.CompletionTime = br.Status.CompletionTime
	run.Status.Conditions = knativev1.Conditions{}

	for _, condition := range br.Status.Conditions {
		log.Printf("Updating Tekton Run with BuildRun: status=%q, reason=%q, message=%q",
			condition.Status, condition.Reason, condition.Message)
		severity := apis.ConditionSeverityInfo
		if condition.Status == corev1.ConditionFalse {
			severity = apis.ConditionSeverityError
		}
		run.Status.Conditions = append(run.Status.Conditions, apis.Condition{
			Type:               apis.ConditionType(string(condition.Type)),
			Status:             condition.Status,
			LastTransitionTime: apis.VolatileTime{Inner: condition.LastTransitionTime},
			Reason:             condition.Reason,
			Message:            condition.Message,
			Severity:           severity,
		})
	}

	if len(run.Status.Conditions) == 0 {
		run.Status.Conditions = []apis.Condition{{
			Type:               apis.ConditionSucceeded,
			Status:             corev1.ConditionUnknown,
			LastTransitionTime: apis.VolatileTime{Inner: metav1.Now()},
		}}
	}

	_, err := c.tektonClientset.TektonV1alpha1().
		Runs(run.Namespace).
		UpdateStatus(c.ctx, run, metav1.UpdateOptions{})
	return err
}

// manageBuildRunForRun inspect the informed Tekton Run object to identify if the respective BuildRun
// has been created. If the BuildRun exists, its status will be copied into the Tekton Run, otherwise
// a new BuildRun instance is created and recorded the on Run's ExtraFields.
func (c *RunController) manageBuildRunForRun(run *tknapisv1alpha1.Run) error {
	if run.IsDone() {
		log.Printf("Tekton Run %q is synchronized! Successful=%v, Canceled=%v",
			run.GetName(), run.IsSuccessful(), run.IsCancelled())
		return nil
	}

	var fields ExtraFields
	err := run.Status.DecodeExtraFields(&fields)
	if err != nil {
		return err
	}

	var br *v1alpha1.BuildRun
	if fields.IsEmpty() {
		br, err = c.createBuildRun(run)
		if err != nil {
			return err
		}
		log.Printf("Dispatching BuildRun %q for Tekton Run %q", br.GetName(), run.GetName())

		// recording the BuildRun name created using ExtraFields
		fields = ExtraFields{BuildRunName: br.GetName()}
		if err = run.Status.EncodeExtraFields(&fields); err != nil {
			return err
		}
		now := metav1.Now()
		run.Status.StartTime = &now
	} else {
		br, err = c.buildRunLister.BuildRuns(run.GetNamespace()).Get(fields.BuildRunName)
		if err != nil {
			return err
		}
		log.Printf("Updating Tekton Run %q status with BuildRun %q", run.GetName(), br.GetName())
	}

	return c.updateRunStatus(run, br)
}

// sync handles Tekton Run resource changes.
func (c *RunController) sync(key string) error {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	log.Printf("Syncing Tekton Run named '%s/%s'...", ns, name)
	run, err := c.runLister.Runs(ns).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return c.manageBuildRunForRun(run)
}

func (c *RunController) processor() {
	for processNextItem(c.wq, c.sync) {
	}
}

// compareBuildRunAndEnqueueRunOwner compares BuildRun objects and enqueue its owner, as long as it's
// owned by Tekton Run resource.
func (c *RunController) compareBuildRunAndEnqueueRunOwner(oldObj, newObj interface{}) {
	oldBR, ok := oldObj.(*v1alpha1.BuildRun)
	if !ok {
		log.Printf("Unable to cast object as Shipwright BuildRun: '%#v'", oldObj)
		return
	}
	newBR, ok := newObj.(*v1alpha1.BuildRun)
	if !ok {
		log.Printf("Unable to cast object as Shipwright BuildRun: '%#v'", newObj)
		return
	}

	if reflect.DeepEqual(oldBR.Status, newBR.Status) {
		return
	}

	runName := searchBuildRunForRunOwner(newBR)
	if runName == nil {
		return
	}
	c.wq.Add(runName.String())
}

// enqueueBuildRunOwner inspect the BuildRun instance in order to determine the Tekton Run owner
// instance, and this Tekton Run instance will be enqueued instead.
func (c *RunController) enqueueBuildRunOwner(obj interface{}) {
	br, ok := obj.(*v1alpha1.BuildRun)
	if !ok {
		log.Printf("Unable to cast object as Shipwright BuildRun: '%#v'", obj)
		return
	}

	runName := searchBuildRunForRunOwner(br)
	if runName == nil {
		log.Printf("BuildRun instance is not owned by Tekton Run: '%#v'", obj)
		return
	}
	log.Printf("BuildRun %q is owned by Tekton Run %q", br.GetName(), runName)

	_, err := c.runLister.Runs(runName.Namespace).Get(runName.Name)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Printf("Error retrieveing Tekton Run: '%#v'", err)
		}
		return
	}
	c.wq.Add(runName.String())
}

// compareAndEnqueueRun inspect the Run instances comparing if both spec or status have been updated,
// and when it happens it enqueues the new object.
func (c *RunController) compareAndEnqueueRun(oldObj, newObj interface{}) {
	c.m.Lock()
	defer c.m.Unlock()

	oldRun, ok := oldObj.(*tknapisv1alpha1.Run)
	if !ok {
		log.Printf("Unable to cast object as Tekton Run: '%#v'", oldObj)
		return
	}
	newRun, ok := newObj.(*tknapisv1alpha1.Run)
	if !ok {
		log.Printf("Unable to cast object as Tekton Run: '%#v'", newObj)
		return
	}

	if reflect.DeepEqual(oldRun.Spec, newRun.Spec) &&
		reflect.DeepEqual(oldRun.Status, newRun.Status) {
		return
	}

	workQueueAdd(c.wq, newObj)
}

// Start the controller by waiting for informer cache synchronization.
func (c *RunController) Start() error {
	log.Printf("Waiting for Tekton Run informer cache synchronization")
	if !cache.WaitForCacheSync(c.ctx.Done(), c.runInformerSynced, c.buildRunInformerSynced) {
		return fmt.Errorf("informers haven't synced")
	}
	return nil
}

// Run the event processor until done channel is issued.
func (c *RunController) Run() error {
	defer c.wq.ShutDown()

	log.Printf("Starting Tekton Run event processor")
	go wait.Until(c.processor, 100*time.Millisecond, c.ctx.Done())

	log.Printf("Tekton Run controller is running!")
	<-c.ctx.Done()
	log.Printf("Tekton Run controller is shutting down..")
	return nil
}

// NewRunController instantiate the Tekton Run controller.
func NewRunController(
	ctx context.Context,
	runInformer tkninformerv1alpha1.Interface,
	tektonClientset tknclientset.Interface,
	buildRunInformer buildinformer.BuildRunInformer,
	buildClientset buildclientset.Interface,
) *RunController {
	wq := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "runs")
	c := &RunController{
		ctx: ctx,

		runInformer:       runInformer,
		runLister:         runInformer.Runs().Lister(),
		runInformerSynced: runInformer.Runs().Informer().HasSynced,
		tektonClientset:   tektonClientset,

		buildRunInformer:       buildRunInformer,
		buildRunLister:         buildRunInformer.Lister(),
		buildRunInformerSynced: buildRunInformer.Informer().HasSynced,
		buildClientset:         buildClientset,

		wq: wq,
	}
	// the Tekton Run objects are filtered by referencing Shipwright resources, but then are simply
	// compared and enqueued regularly
	runInformer.Runs().Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: tkncontroller.FilterRunRef(ShipwrightAPIVersion, "Build"),
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    enqueueRunFn(wq),
			UpdateFunc: c.compareAndEnqueueRun,
			DeleteFunc: enqueueRunFn(wq),
		},
	})
	// the BuildRun objects are filtered by the ones owned by Tekton, and therefore, on enqueuing
	// those objects the actual Tekton Run name is extracted and enqueued instead
	buildRunInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: filterBuildRunOwnedByRun,
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc:    c.enqueueBuildRunOwner,
			UpdateFunc: c.compareBuildRunAndEnqueueRunOwner,
			DeleteFunc: c.enqueueBuildRunOwner,
		},
	})
	return c
}
