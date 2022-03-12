package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/otaviof/shipwright-trigger/pkg/trigger/clients"
	"github.com/otaviof/shipwright-trigger/pkg/trigger/inventory"
	"github.com/otaviof/shipwright-trigger/test/stubs"
	buildclientset "github.com/shipwright-io/build/pkg/client/clientset/versioned"
	tknclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tkninformers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// newTestPipelineRunController creates a new test instance of the PipelineRunController, already
// started and ready to process PipelineRun objects.
func newTestPipelineRunController(
	t *testing.T,
	ctx context.Context,
	tektonClientset tknclientset.Interface,
	buildClientset buildclientset.Interface,
	buildInventory inventory.Interface,
) Interface {
	g := gomega.NewWithT(t)

	informerFactory := tkninformers.NewSharedInformerFactory(tektonClientset, 0)
	tektonInfomer := informerFactory.Tekton().V1beta1()

	c := NewPipelineRunController(
		ctx,
		tektonInfomer,
		tektonClientset,
		buildClientset,
		buildInventory,
	)

	informerFactory.Start(ctx.Done())
	err := c.Start()
	g.Expect(err).To(gomega.BeNil())

	go func() {
		err := c.Run()
		g.Expect(err).To(gomega.BeNil())
	}()
	return c
}

// assertBuildRunListLenEventually assert the amount of BuildRun instances matches what's expected.
func assertBuildRunListLenEventually(
	t *testing.T,
	ctx context.Context,
	buildClientset buildclientset.Interface,
	expectedLen int,
) {
	g := gomega.NewWithT(t)

	// when zero is expected, lets give some graceful time to wait for the controller event loop,
	// in case the workqueue contains left over objects
	if expectedLen == 0 {
		time.Sleep(5 * time.Second)
	}

	g.Eventually(func() int {
		buildRuns, err := buildClientset.ShipwrightV1alpha1().
			BuildRuns(stubs.Namespace).
			List(ctx, metav1.ListOptions{})
		if err != nil {
			return -1
		}
		return len(buildRuns.Items)
	}).Should(gomega.Equal(expectedLen))
}

// TestNewPipelineRunController asserts the primary workflow of the PipelineRunController is working,
// it checks different PipelineRun instances including Custom-Tasks.
func TestNewPipelineRunController(t *testing.T) {
	g := gomega.NewWithT(t)

	ctx := context.Background()
	fakeKubeClients := clients.NewFakeKubeClients()
	tektonClientset, _ := fakeKubeClients.GetTektonClientset()
	buildClientset, _ := fakeKubeClients.GetShipwrightClientset()
	fakeBuildInventory := inventory.NewFakeInventory()

	_ = newTestPipelineRunController(t, ctx, tektonClientset, buildClientset, fakeBuildInventory)

	// asserting the PipelineRunController won't process an incomplete instance, in this test case
	// the instance does not have any status set
	t.Run("no status recorded on pipelinerun instance", func(t *testing.T) {
		pipelineRun := stubs.TektonPipelineRun("empty")

		_, err := tektonClientset.TektonV1beta1().
			PipelineRuns(stubs.Namespace).
			Create(ctx, &pipelineRun, metav1.CreateOptions{})
		g.Expect(err).To(gomega.BeNil())

		assertBuildRunListLenEventually(t, ctx, buildClientset, 0)
	})

	// asserting the PipelineRunController won't process a Custom-Tasks instance, in this case it
	// will refuse to process objects referring back to Shipwright resources
	t.Run("custom-task pipelinerun instance", func(t *testing.T) {
		pipelineRun := stubs.TektonPipelineRunSucceeded("custom-task")
		pipelineRun.Status.PipelineSpec = stubs.TektonPipelineRunStatusCustomTaskShipwright

		_, err := tektonClientset.TektonV1beta1().
			PipelineRuns(stubs.Namespace).
			Create(ctx, &pipelineRun, metav1.CreateOptions{})
		g.Expect(err).To(gomega.BeNil())

		assertBuildRunListLenEventually(t, ctx, buildClientset, 0)
	})

	// asserting the PipelineRunController will skip instances with the PipelineRunNameKey set to the
	// same object name, the label is added to avoid reprocessing instances
	t.Run("name label present on pipelinerun instance", func(t *testing.T) {
		pipelineRun := stubs.TektonPipelineRunSucceeded("labeled")
		pipelineRun.SetLabels(map[string]string{
			PipelineRunNameKey: pipelineRun.GetName(),
		})

		_, err := tektonClientset.TektonV1beta1().
			PipelineRuns(stubs.Namespace).
			Create(ctx, &pipelineRun, metav1.CreateOptions{})
		g.Expect(err).To(gomega.BeNil())

		assertBuildRunListLenEventually(t, ctx, buildClientset, 0)
	})

	// asserting the PipelineRunController will process the complete instance informed, it's set to
	// "Succeeded" status, and therefore will trigger a new BuildRun instance. The test also asserts
	// the PipelineRun instance got the PipelineRunNameKey label
	t.Run("complete pipelinerun instance", func(t *testing.T) {
		build := stubs.ShipwrightBuild("name")
		fakeBuildInventory.Add(&build)

		pipelineRun := stubs.TektonPipelineRunSucceeded("complete")

		_, err := tektonClientset.TektonV1beta1().
			PipelineRuns(stubs.Namespace).
			Create(ctx, &pipelineRun, metav1.CreateOptions{})
		g.Expect(err).To(gomega.BeNil())

		assertBuildRunListLenEventually(t, ctx, buildClientset, 1)

		g.Eventually(func() bool {
			pr, err := tektonClientset.TektonV1beta1().
				PipelineRuns(pipelineRun.GetNamespace()).
				Get(ctx, pipelineRun.GetName(), metav1.GetOptions{})
			if err != nil {
				return false
			}
			return pipelineRunNameMatchesLabel(pr)
		}).Should(gomega.BeTrue())
	})
}
