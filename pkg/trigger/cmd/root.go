package cmd

import (
	"log"
	"time"

	"github.com/otaviof/shipwright-trigger/pkg/trigger/clients"
	"github.com/otaviof/shipwright-trigger/pkg/trigger/controllers"
	"github.com/otaviof/shipwright-trigger/pkg/trigger/inventory"
	"github.com/otaviof/shipwright-trigger/pkg/trigger/webhooks"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// configFlags flags for the Kubernetes clients.
var configFlags = genericclioptions.NewConfigFlags(true)

// rootCmd cobra command definition for the Shipwright Trigger application.
var rootCmd = &cobra.Command{
	Use:  "trigger",
	RunE: runE,
}

func init() {
	flagSet := rootCmd.Flags()
	configFlags.AddFlags(flagSet)
}

// runE instantiate the whole application, by loading the Kubernetes clients first and then loading
// Kubernetes controller instances. By last the WebHook server is instantiated.
func runE(cmd *cobra.Command, _ []string) error {
	kubeClients, err := clients.NewKubeClients(configFlags)
	if err != nil {
		return err
	}
	buildInventory := inventory.NewInventory()

	// instantiating all controllers at once
	c, err := controllers.NewController(cmd.Context(), kubeClients, 10*time.Minute, buildInventory)
	if err != nil {
		return err
	}
	// starting the informers, waiting for cache synchronization
	if err := c.Start(); err != nil {
		return err
	}
	// running the controller event loop processors in background
	go func() {
		if err := c.Run(); err != nil {
			log.Fatal(err)
		}
	}()

	// listening for the webhook requests
	httpServer, err := webhooks.NewHTTPServer(cmd.Context(), kubeClients, buildInventory)
	if err != nil {
		return err
	}
	return httpServer.Listen(":8080")
}

// NewRootCmd exposes the cobra command instance.
func NewRootCmd() *cobra.Command {
	return rootCmd
}
