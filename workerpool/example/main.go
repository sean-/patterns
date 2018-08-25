package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sean-/patterns/workerpool"
	"github.com/sean-/seed"
	"github.com/sean-/sysexits"
)

func main() {
	os.Exit(realMain())
}

type task struct {
}

type producer struct {
	queue workerpool.SubmissionQueue

	created uint
	stalls  uint
}

func (p *producer) IncrCreated() {
	p.created++
}

func (p *producer) IncrStalls() {
	p.stalls++
}

func (p *producer) Run(ctx context.Context, tid workerpool.ThreadID) error {
	remainingWork := 2
	var fullCount int

STOP_PRODUCING_WORK:
	for remainingWork > 0 {
		d := 1 * time.Second
		t := &task{}
		select {
		case p.queue <- t:
			p.IncrCreated()
			remainingWork--
			fmt.Printf("producer[%d]: added a work item: %d remaining\n", tid, remainingWork)
		case <-ctx.Done():
			fmt.Printf("producer[%d]: shutting down\n", tid)
			break STOP_PRODUCING_WORK
		default:
			fmt.Printf("unable to add an item to the work queue, it's full: %d\n", len(p.queue))
			p.IncrStalls()
		}
		//fmt.Printf("producer[%d]: %d Sleeping for %s\n", tid, i, d)
		time.Sleep(d)
		//fmt.Printf("producer[%d]: %d Done sleeping\n", tid, i)
	}
	fmt.Printf("producer[%d]: exiting: queue full %d times\n", tid, fullCount)

	return nil
}

type producerFactory struct {
}

func (pf *producerFactory) New(q workerpool.SubmissionQueue) (workerpool.Producer, error) {
	return &producer{queue: q}, nil
}

type worker struct {
	queue workerpool.SubmissionQueue
}

func newWorker(q workerpool.SubmissionQueue) *worker {
	return &worker{queue: q}
}

func (w *worker) Run(ctx context.Context, tid workerpool.ThreadID) error {
	var workDone int
	var workStall int

CHANNEL_CLOSED:
	for {
		select {
		case t, ok := <-w.queue:
			_ = t
			if !ok {
				break CHANNEL_CLOSED
			}
			workDone++
		case <-ctx.Done():
			fmt.Printf("worker[%d]: shutting down\n", tid)
			break CHANNEL_CLOSED
		default:
			workStall++
		}

		d := 2 * time.Second
		fmt.Printf("worker[%d]: Sleeping for %s\n", tid, d)
		time.Sleep(d)
		fmt.Printf("worker[%d]: Done sleeping\n", tid)
	}
	fmt.Printf("worker[%d]: exiting: completed %d, stalled %d times\n", tid, workDone, workStall)

	return nil
}

type workerFactory struct {
}

func (wf *workerFactory) New(q workerpool.SubmissionQueue) (workerpool.Worker, error) {
	return &worker{queue: q}, nil
}

func realMain() int {
	seed.MustInit()

	app := workerpool.New(
		workerpool.Config{
			InitialNumWorkers:   1,
			InitialNumProducers: 5,
			WorkQueueDepth:      2,
		},
		workerpool.Handlers{
			Reload: nil,
		},
		workerpool.Threads{
			ProducerFactory: &producerFactory{},
			WorkerFactory:   &workerFactory{},
		},
	)
	defer app.Stop()

	if err := runSignalHandler(app); err != nil {
		fmt.Printf("unable to launch signal handler: %v", err)
		return sysexits.Software
	}

	if err := app.StartProducers(); err != nil {
		fmt.Printf("unable to start the workerpool producers: %v", err)
		return sysexits.Software
	}
	fmt.Println("started producers")

	if err := app.StartWorkers(); err != nil {
		fmt.Printf("unable to start the workerpool workers: %v", err)
		return sysexits.Software
	}
	fmt.Println("started workers")

	if err := app.WaitProducers(); err != nil {
		fmt.Printf("error waiting for workpool producers: %v", err)
		return sysexits.Software
	}
	fmt.Println("finished waiting for producers")

	if err := app.WaitWorkers(); err != nil {
		fmt.Printf("error waiting for the workerpool to drain: %v", err)
		return sysexits.Software
	}
	fmt.Println("finished waiting for workers")

	return sysexits.OK
}

func runSignalHandler(wp workerpool.Interface) error {
	sigCh := make(chan os.Signal, 5)

	signal.Notify(sigCh,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGTERM,
	)

	// Spin off a signal handler
	go func() {
		for {
			select {
			case <-wp.ShutdownCtx().Done():
				fmt.Println("shutdown initiated, terminating signal handler")
				wp.InitiateShutdown()
				return
			case sig := <-sigCh:
				switch sig {
				case syscall.SIGHUP:
					fmt.Printf("received %s, reloading\n", sig)
					wp.Reload()
				case syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM:
					fmt.Printf("received %s, shutting down\n", sig)
					wp.InitiateShutdown()
					return
				}
			}
		}
	}()

	return nil
}
