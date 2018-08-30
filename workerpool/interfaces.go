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

	StartProducers() error
	StartConsumers() error
	WaitProducers() error
	WaitConsumers() error
}

// Factories is a container for different types of factories.
type Factories struct {
	ProducerFactory ProducerFactory
	ConsumerFactory ConsumerFactory
}

// Handlers is a container for various types of handler functions called
// throughout the lifecycle of a workerpool.
type Handlers struct {
	ReloadFunc   func()
	ShutdownFunc func()
	ShutdownCtx  context.Context

	ProducerFactoryNewErr func(error)
	ConsumerFactoryNewErr func(error)

	ProducerRunErr func(error) (resumable bool)
	ConsumerRunErr func(error) (resumable bool)
}

// ThreadID is a generic identifier for a given thread.
type ThreadID int

// Task represents a logical unit of work.
type Task interface {
}

// SubmissionQueue is a channel which facilitates work tasks.
type SubmissionQueue chan Task

// Producer is the producer of work across a pool of workers.
type Producer interface {
	Run(context.Context, ThreadID) error
}

// ProducerFactory handles the construction of a new Producer.
type ProducerFactory interface {
	New(SubmissionQueue) (Producer, error)
	Finished(ThreadID, Producer)
}

// Consumer executes a logical unit of work tasked by the Producer.
type Consumer interface {
	Run(context.Context, ThreadID) error
}

// ConsumerFactory handles the construction of a new Worker.
type ConsumerFactory interface {
	New(SubmissionQueue) (Consumer, error)
	Finished(ThreadID, Consumer)
}
