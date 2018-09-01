package workerpool

import (
	"context"
	"fmt"
	"sync"

	"github.com/mohae/deepcopy"
)

// Config is the initial configuration of the workerpool
type Config struct {
	InitialNumProducers uint
	InitialNumConsumers uint
	WorkQueueDepth      uint
}

// workerPool is a private type that can only be created via New
type workerPool struct {
	submissionQueue SubmissionQueue

	producersWG *sync.WaitGroup
	consumersWG *sync.WaitGroup

	signalLock   sync.Mutex
	shutdownFunc func()
	shutdownCtx  context.Context
	shutdown     bool

	cfg       Config
	factories Factories
	handlers  Handlers
}

// New creates a new workerpool populated with Config, Handlers, and Factories.
func New(appCfg Config, factories Factories, handlers Handlers) *workerPool {
	app := &workerPool{
		submissionQueue: make(SubmissionQueue, appCfg.WorkQueueDepth),
		producersWG:     &sync.WaitGroup{},
		consumersWG:     &sync.WaitGroup{},

		shutdownFunc: handlers.ShutdownFunc,
		shutdownCtx:  handlers.ShutdownCtx,

		cfg:       deepcopy.Copy(appCfg).(Config),
		factories: factories,
		handlers:  deepcopy.Copy(handlers).(Handlers),
	}

	return app
}

// InitiateShutdown starts the shutdown process if a shutdown has not already
// been initiated.  Returns true if the shutdown was started.
func (a *workerPool) InitiateShutdown() (bool, error) {
	a.signalLock.Lock()
	defer a.signalLock.Unlock()

	if a.shutdown {
		return false, nil
	}

	a.shutdownFunc()
	a.shutdown = true

	return true, nil
}

// Reload calls the reload handler, if set
func (a *workerPool) Reload() {
	a.signalLock.Lock()
	defer a.signalLock.Unlock()

	if a.handlers.ReloadFunc != nil {
		a.handlers.ReloadFunc()
	}
}

// ShutdownCtx returns the shutdown context for the workerpool (shared between
// producers and workers).
func (a *workerPool) ShutdownCtx() context.Context {
	return a.shutdownCtx
}

// StartProducers spawns InitialNumProducers and calls Producer.Run() for each
// Producer.  If Producer.Run() returns an error, Handlers.ProducerRunErr() is
// called.  If ProducerRunErr() is nil, panic() is called instead.
func (a *workerPool) StartProducers() error {
	for i := a.cfg.InitialNumProducers; i > 0; i-- {
		a.producersWG.Add(1)
		go func(i uint) {
			defer a.producersWG.Done()
			threadID := ThreadID(i)
			producer, err := a.factories.ProducerFactory.New(threadID, a.submissionQueue)
			if err != nil {
				if a.handlers.ProducerFactoryNewErr == nil {
					panic(fmt.Sprintf("error creating a new producer %d: %v", i, err))
				}

				a.handlers.ProducerFactoryNewErr(err)
			}

			if err := producer.Run(a.shutdownCtx); err != nil {
				if a.handlers.ProducerRunErr == nil {
					panic(fmt.Sprintf("error starting producer thread %d: %v", i, err))
				}

				if resume := a.handlers.ProducerRunErr(err); !resume {
					return
				}
			}

			a.factories.ProducerFactory.Finished(threadID, producer)
		}(i)
	}

	return nil
}

// StartConsumers starts the worker pool
func (a *workerPool) StartConsumers() error {
	for i := a.cfg.InitialNumConsumers; i > 0; i-- {
		a.consumersWG.Add(1)
		go func(i uint) {
			defer a.consumersWG.Done()
			threadID := ThreadID(i)
			consumer, err := a.factories.ConsumerFactory.New(threadID, a.submissionQueue)
			if err != nil {
				if a.handlers.ConsumerFactoryNewErr == nil {
					panic(fmt.Sprintf("error creating a new consumer %d: %v", i, err))
				}

				a.handlers.ConsumerFactoryNewErr(err)
			}

			if err := consumer.Run(a.shutdownCtx); err != nil {
				if a.handlers.ConsumerRunErr == nil {
					panic(fmt.Sprintf("error starting consumer thread %d: %v", i, err))
				}

				if resume := a.handlers.ConsumerRunErr(err); !resume {
					return
				}
			}

			a.factories.ConsumerFactory.Finished(threadID, consumer)
		}(i)
	}

	return nil
}

// WaitProducers blocks until all producers have exited or ShutdownCtx has been
// closed.  WaitProducers closes the submission queue.
func (a *workerPool) WaitProducers() error {
	a.producersWG.Wait()
	close(a.submissionQueue)

	return nil
}

// WaitProducers blocks until all producers have exited or ShutdownCtx has been
// closed.
func (a *workerPool) WaitConsumers() error {
	a.consumersWG.Wait()

	return nil
}
