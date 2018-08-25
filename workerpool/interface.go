package workerpool

import "context"

type Interface interface {
	ShutdownCtx() context.Context
	InitiateShutdown() (bool, error)
	Reload()
}
