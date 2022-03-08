package controllers

// Interface defines the common elements for the controllers on this package.
type Interface interface {
	// Start the controller by waiting for the informer(s) cache synchronization.
	Start() error

	// Run the controller by starting the event loop and waiting for done channel.
	Run() error
}
