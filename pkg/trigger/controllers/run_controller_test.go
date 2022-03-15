package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/otaviof/shipwright-trigger/pkg/trigger/clients"
	"github.com/otaviof/shipwright-trigger/test/stubs"
	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"
	buildclientset "github.com/shipwright-io/build/pkg/client/clientset/versioned"
	buildinformers "github.com/shipwright-io/build/pkg/client/informers/externalversions"
	tknapisv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tknclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tkninformers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions"
	tkninformerv1alpha1 "github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// newTestRunController instantiate a RunController for testing, sharing the controller instance and
// its Tekton informer.
func newTestRunController(
	t *testing.T,
	ctx context.Context,
	tektonClientset tknclientset.Interface,
	buildClientset buildclientset.Interface,
) (Interface, tkninformerv1alpha1.Interface) {
	g := gomega.NewWithT(t)

	tektonInformerFactory := tkninformers.NewSharedInformerFactory(tektonClientset, 0)
	tektonInfomer := tektonInformerFactory.Tekton().V1alpha1()

	buildInformerFactory := buildinformers.NewSharedInformerFactory(buildClientset, 0)
	buildInformer := buildInformerFactory.Shipwright().V1alpha1().BuildRuns()

	c := NewRunController(ctx, tektonInfomer, tektonClientset, buildInformer, buildClientset)

	tektonInformerFactory.Start(ctx.Done())
	buildInformerFactory.Start(ctx.Done())

	err := c.Start()
	g.Expect(err).To(gomega.BeNil())

	go func() {
		err := c.Run()
		g.Expect(err).To(gomega.BeNil())
	}()
	return c, tektonInfomer
}

// TestNewRunController asserts the primary workflows of the RunController, first Run instances not
// pointing Shipwright should be ignored by the controller, and Shipwright references should be
// subject to the business logic.
func TestNewRunController(t *testing.T) {
	g := gomega.NewWithT(t)

	ctx := context.Background()
	_, cancelCtxFn := context.WithCancel(ctx)

	fakeKubeClients := clients.NewFakeKubeClients()
	tektonClientset, _ := fakeKubeClients.GetTektonClientset()
	buildClientset, _ := fakeKubeClients.GetShipwrightClientset()

	_, tektonInformer := newTestRunController(t, ctx, tektonClientset, buildClientset)

	// asserting no BuildRun objects will be created when a Run instance not referring to Shipwright
	// objects is created, the Run instance shall be ignored by the RunController without returning
	// errors
	t.Run("run instance not pointing to shipwright", func(t *testing.T) {
		run := stubs.TektonRun("internal-task", stubs.TektonTaskRefToTekton)

		_, err := tektonClientset.TektonV1alpha1().
			Runs(stubs.Namespace).
			Create(ctx, &run, metav1.CreateOptions{})
		g.Expect(err).To(gomega.BeNil())

		assertBuildRunListLenEventually(t, ctx, buildClientset, 0)
	})

	// run instance name with reference to shipwright resource
	runName := "shipwright-task"

	// buildrun instance employed in subsequent tests
	var buildRun *v1alpha1.BuildRun

	// asserting a BuildRun is produced when a new Run instance pointing to Shipwright is created,
	// checks if the BuildRun attributes are as expected, and checks if the parameters in the Run
	// instance are ported to the BuildRun
	t.Run("run instance pointing to shipwright produces a buildrun", func(t *testing.T) {
		run := stubs.TektonRun(runName, stubs.TektonTaskRefToShipwright)
		run.Spec.Params = []tknapisv1beta1.Param{{
			Name:  "key",
			Value: *tknapisv1beta1.NewArrayOrString("value"),
		}}

		_, err := tektonClientset.TektonV1alpha1().
			Runs(stubs.Namespace).
			Create(ctx, &run, metav1.CreateOptions{})
		g.Expect(err).To(gomega.BeNil())

		assertBuildRunListLenEventually(t, ctx, buildClientset, 1)

		buildRuns, err := buildClientset.ShipwrightV1alpha1().
			BuildRuns(stubs.Namespace).
			List(ctx, metav1.ListOptions{})
		g.Expect(err).To(gomega.BeNil())

		buildRun = &buildRuns.Items[0]
		g.Expect(buildRun).NotTo(gomega.BeNil())
		g.Expect(buildRun.Spec.BuildRef.Name).
			To(gomega.Equal(stubs.TektonTaskRefToShipwright.Name))
		g.Expect(buildRun.Spec.ParamValues).
			To(gomega.Equal(TektonRunParamsToShipwrightParamValues(&run)))
	})

	// asserting the Run instance ExtraFields is updated with the reference to the BuildRun instance
	// created on the previous step.
	t.Run("run instance extra-field status is updated", func(_ *testing.T) {
		g.Eventually(func() bool {
			run, err := tektonInformer.Runs().Lister().Runs(stubs.Namespace).Get(runName)
			if err != nil {
				return false
			}

			var fields ExtraFields
			if err = run.Status.DecodeExtraFields(&fields); err != nil {
				return false
			}
			return fields.BuildRunName == buildRun.GetName()
		}).Should(gomega.BeTrue())
	})

	// asserting the Run instance will get the status updates applied to the BuildRun, this resource
	// status is updated, and it should eventually land on the parent (Run)
	t.Run("buildrun status is reflected on run instance", func(_ *testing.T) {
		buildRun.Status.Conditions = v1alpha1.Conditions{{
			Type:               v1alpha1.Succeeded,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "reason",
			Message:            "message",
		}}

		var err error
		buildRun, err = buildClientset.ShipwrightV1alpha1().
			BuildRuns(stubs.Namespace).
			UpdateStatus(ctx, buildRun, metav1.UpdateOptions{})
		g.Expect(err).To(gomega.BeNil())

		// in order to avoid racing condition with the Controller workflow, the context is cancelled
		// in order to stop the event processor, plus graceful sleep
		cancelCtxFn()
		time.Sleep(5 * time.Second)

		// comparing the run instance status with the status updates applied to the BuildRun
		g.Eventually(func() bool {
			run, err := tektonInformer.Runs().Lister().Runs(stubs.Namespace).Get(runName)
			if err != nil {
				return false
			}

			conditions := run.Status.GetConditions()
			if len(conditions) == 0 {
				return false
			}
			condition := conditions[0]

			return string(condition.Type) == string(v1alpha1.Succeeded) &&
				string(condition.Status) == string(corev1.ConditionTrue) &&
				condition.Reason == "reason" &&
				condition.Message == "message"
		}).Should(gomega.BeTrue())
	})
}
