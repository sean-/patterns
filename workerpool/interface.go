package workerpool

import "context"

// Interface is the workerpool interface that is exported
type Interface interface {
	// ShutdownCtx is the shutdown context, used to poll for the status of shutdown
	ShutdownCtx() context.Context

	// InitiateShutdown cancels the context used to signal its time to shut down
	InitiateShutdown() (bool, error)

	// Reload is a configuration reload handler (e.g. for use with SIGHUP)
	Reload()
}
