package workerpool

import (
	"context"
	"fmt"
	"sync"
)

type workerPool struct {
	submissionQueue SubmissionQueue

	producersWG *sync.WaitGroup
	workersWG   *sync.WaitGroup

	signalLock  sync.Mutex
	shutdownFn  func()
	shutdownCtx context.Context
	shutdown    bool

	cfg      Config
	handlers Handlers
	threads  Threads
}

func New(appCfg Config, handlers Handlers, threads Threads) *workerPool {
	ctx, cancel := context.WithCancel(context.Background())
	app := &workerPool{
		submissionQueue: make(SubmissionQueue, appCfg.WorkQueueDepth),
		producersWG:     &sync.WaitGroup{},
		workersWG:       &sync.WaitGroup{},

		shutdownFn:  cancel,
		shutdownCtx: ctx,

		cfg:      appCfg.Copy(),
		handlers: handlers.Copy(),
		threads:  threads.Copy(),
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

	a.shutdownFn()
	a.shutdown = true

	return true, nil
}

func (a *workerPool) Reload() {
	a.signalLock.Lock()
	defer a.signalLock.Unlock()

	if a.handlers.Reload != nil {
		a.handlers.Reload()
	}
}

func (a *workerPool) StartWorkers() error {
	for i := a.cfg.InitialNumWorkers; i > 0; i-- {
		a.workersWG.Add(1)
		go func(i uint) {
			defer a.workersWG.Done()
			threadID := ThreadID(i)
			worker, err := a.threads.WorkerFactory.New(a.submissionQueue)
			if err != nil {
				if a.handlers.WorkerFactoryNewErr == nil {
					panic(fmt.Sprintf("error creating a new worker %d: %v", i, err))
				}

				a.handlers.WorkerFactoryNewErr(err)
			}

			if err := worker.Run(a.shutdownCtx, threadID); err != nil {
				if a.handlers.WorkerRunErr == nil {
					panic(fmt.Sprintf("error starting worker thread %d: %v", i, err))
				}

				if resume := a.handlers.WorkerRunErr(err); !resume {
					return
				}
			}

			a.threads.WorkerFactory.Finished(threadID, worker)
		}(i)
	}

	return nil
}

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
			producer, err := a.threads.ProducerFactory.New(a.submissionQueue)
			if err != nil {
				if a.handlers.ProducerFactoryNewErr == nil {
					panic(fmt.Sprintf("error creating a new producer %d: %v", i, err))
				}

				a.handlers.ProducerFactoryNewErr(err)
			}

			if err := producer.Run(a.shutdownCtx, threadID); err != nil {
				if a.handlers.ProducerRunErr == nil {
					panic(fmt.Sprintf("error starting producer thread %d: %v", i, err))
				}

				if resume := a.handlers.ProducerRunErr(err); !resume {
					return
				}
			}

			a.threads.ProducerFactory.Finished(threadID, producer)
		}(i)
	}

	return nil
}

func (a *workerPool) Stop() error {
	a.InitiateShutdown()

	return nil
}

func (a *workerPool) WaitProducers() error {
	a.producersWG.Wait()
	close(a.submissionQueue)

	return nil
}

func (a *workerPool) WaitWorkers() error {
	a.workersWG.Wait()

	return nil
}
