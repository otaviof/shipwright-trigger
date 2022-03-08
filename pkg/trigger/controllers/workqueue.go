package controllers

import (
	"log"
	"reflect"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	tknapisv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tknapisv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// syncFn the controller sync function signature.
type syncFn func(string) error

// enqueueFn the event added or removed handler function signature.
type enqueueFn func(interface{})

// compareAndEnqueueFn the updated event handler function signature.
type compareAndEnqueueFn func(interface{}, interface{})

// processNextItem executes the sync function handling possible errors, rate limiting and the
// workqueue instance.
func processNextItem(wq workqueue.RateLimitingInterface, fn syncFn) bool {
	obj, shutdown := wq.Get()
	if shutdown {
		return false
	}
	err := func(obj interface{}) error {
		defer wq.Done(obj)

		key, ok := obj.(string)
		if !ok {
			wq.Forget(obj)
			log.Printf("Expected string on the workqueue, instead it contains: '%#v'", obj)
			return nil
		}

		if err := fn(key); err != nil {
			wq.AddRateLimited(obj)
			return err
		}
		wq.Forget(obj)
		return nil
	}(obj)
	if err != nil {
		log.Printf("Error processing item: %q", err.Error())
	}
	return true
}

// workQueueAdd adds the informed object on the workqueue.
func workQueueAdd(wq workqueue.RateLimitingInterface, obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Printf("Error inspecting object: '%#v'", err)
		return
	}
	wq.Add(key)
}

// enqueueBuildFn enqueues a Shipwright Build object.
func enqueueBuildFn(wq workqueue.RateLimitingInterface) enqueueFn {
	return func(obj interface{}) {
		_, ok := obj.(*v1alpha1.Build)
		if !ok {
			log.Printf("Unable to cast object as Shipwright Build: '%#v'", obj)
			return
		}
		workQueueAdd(wq, obj)
	}
}

// enqueuePipelineRunFn enqueues a Tekton PipelineRun object.
func enqueuePipelineRunFn(wq workqueue.RateLimitingInterface) enqueueFn {
	return func(obj interface{}) {
		_, ok := obj.(*tknapisv1beta1.PipelineRun)
		if !ok {
			log.Printf("Unable to cast object as Tekton PipelineRun: '%#v'", obj)
			return
		}
		workQueueAdd(wq, obj)
	}
}

// enqueueRunFn enqueues a Tekton Run object.
func enqueueRunFn(wq workqueue.RateLimitingInterface) enqueueFn {
	return func(obj interface{}) {
		_, ok := obj.(*tknapisv1alpha1.Run)
		if !ok {
			log.Printf("Unable to cast object as Tekton Run: '%#v'", obj)
			return
		}
		workQueueAdd(wq, obj)
	}
}

// compareAndEnqueueBuildFn compares and enqueue Shipwright Build objects.
func compareAndEnqueueBuildFn(wq workqueue.RateLimitingInterface) compareAndEnqueueFn {
	return func(oldObj, newObj interface{}) {
		oldBuild, ok := oldObj.(*v1alpha1.Build)
		if !ok {
			log.Printf("Unable to cast object as Shipwright Build: '%#v'", oldBuild)
			return
		}
		newBuild, ok := newObj.(*v1alpha1.Build)
		if !ok {
			log.Printf("Unable to cast object as Shipwright Build: '%#v'", newBuild)
			return
		}

		if reflect.DeepEqual(oldBuild.Spec.Source, newBuild.Spec.Source) &&
			reflect.DeepEqual(oldBuild.Spec.Trigger, newBuild.Spec.Trigger) {
			return
		}

		workQueueAdd(wq, newObj)
	}
}

// compareAndEnqueuePipelineRunFn compares and enqueue Tekton PipelineRun objects.
func compareAndEnqueuePipelineRunFn(wq workqueue.RateLimitingInterface) compareAndEnqueueFn {
	return func(oldObj, newObj interface{}) {
		oldRun, ok := oldObj.(*tknapisv1beta1.PipelineRun)
		if !ok {
			log.Printf("Unable to cast object as Tekton Run: '%#v'", oldObj)
			return
		}
		newRun, ok := newObj.(*tknapisv1beta1.PipelineRun)
		if !ok {
			log.Printf("Unable to cast object as Tekton Run: '%#v'", newObj)
			return
		}

		if reflect.DeepEqual(oldRun.Status, newRun.Status) {
			return
		}

		workQueueAdd(wq, newObj)
	}
}

// compareAndEnqueueRunFn compares and enqueues Tekton Run objects.
func compareAndEnqueueRunFn(wq workqueue.RateLimitingInterface) compareAndEnqueueFn {
	return func(oldObj, newObj interface{}) {
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

		workQueueAdd(wq, newObj)
	}
}
