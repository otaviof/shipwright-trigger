package main

import (
	"log"
	"os"

	"github.com/otaviof/shipwright-trigger/pkg/trigger/cmd"
)

func main() {
	if err := cmd.NewRootCmd().Execute(); err != nil {
		log.Fatalf("ERROR: %v", err)
		os.Exit(1)
	}
}
