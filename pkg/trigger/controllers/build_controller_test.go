package controllers

import (
	"context"
	"testing"

	"github.com/onsi/gomega"
	"github.com/otaviof/shipwright-trigger/pkg/trigger/clients"
	"github.com/otaviof/shipwright-trigger/pkg/trigger/inventory"
	"github.com/otaviof/shipwright-trigger/test/stubs"
	buildclientset "github.com/shipwright-io/build/pkg/client/clientset/versioned"
	buildinformers "github.com/shipwright-io/build/pkg/client/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const buildName = "name"

// newTestBuildController creates a new test instance of the BuildController, started and ready to
// process Build objects.
func newTestBuildController(
	t *testing.T,
	ctx context.Context,
	buildClientset buildclientset.Interface,
	buildInventory inventory.Interface,
) Interface {
	g := gomega.NewWithT(t)

	informerFactory := buildinformers.NewSharedInformerFactory(buildClientset, 0)
	buildsInformer := informerFactory.Shipwright().V1alpha1().Builds()

	c := NewBuildController(ctx, buildsInformer, buildInventory)

	informerFactory.Start(ctx.Done())
	err := c.Start()
	g.Expect(err).To(gomega.BeNil())

	go func() {
		err := c.Run()
		g.Expect(err).To(gomega.BeNil())
	}()
	return c
}

// TestNewBuildController asserts the primary BuildController workflow, it adds and removes a Build
// instances from the inventory through the controller event loop.
func TestNewBuildController(t *testing.T) {
	g := gomega.NewWithT(t)

	ctx := context.Background()
	fakeKubeClients := clients.NewFakeKubeClients()
	buildClientset, _ := fakeKubeClients.GetShipwrightClientset()
	fakeBuildInventory := inventory.NewFakeInventory()

	_ = newTestBuildController(t, ctx, buildClientset, fakeBuildInventory)

	// creates a new Build instnance and asserts if the fake Inventory contains the Build on cache
	t.Run("create build instance", func(_ *testing.T) {
		g.Expect(fakeBuildInventory.Contains(buildName)).To(gomega.BeFalse())

		b := stubs.ShipwrightBuild(buildName)
		_, err := buildClientset.ShipwrightV1alpha1().
			Builds(stubs.Namespace).
			Create(ctx, &b, metav1.CreateOptions{})
		g.Expect(err).To(gomega.BeNil())

		g.Eventually(func() bool {
			return fakeBuildInventory.Contains(buildName)
		}).Should(gomega.BeTrue())
	})

	// deletes the Build instance created before, and asserts if the fake Inventory does not contain
	// the instance anymore
	t.Run("delete build instance", func(_ *testing.T) {
		g.Expect(fakeBuildInventory.Contains(buildName)).To(gomega.BeTrue())

		err := buildClientset.ShipwrightV1alpha1().
			Builds(stubs.Namespace).
			Delete(ctx, buildName, metav1.DeleteOptions{})
		g.Expect(err).To(gomega.BeNil())

		g.Eventually(func() bool {
			return fakeBuildInventory.Contains(buildName)
		}).Should(gomega.BeFalse())
	})
}
