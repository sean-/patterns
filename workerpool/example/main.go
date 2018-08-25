package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
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
	queue   workerpool.SubmissionQueue
	created uint64
	stalls  uint64
}

func (p *producer) Run(ctx context.Context, tid workerpool.ThreadID) error {
	const desiredWorkCount = 10
	remainingWork := desiredWorkCount
	var fullCount int

STOP_PRODUCING_WORK:
	for remainingWork > 0 {
		d := 1 * time.Second
		t := &task{}
		select {
		case p.queue <- t:
			p.created++
			remainingWork--
			//fmt.Printf("producer[%d]: added a work item: %d remaining\n", tid, remainingWork)
		case <-ctx.Done():
			//fmt.Printf("producer[%d]: shutting down\n", tid)
			break STOP_PRODUCING_WORK
		default:
			//fmt.Printf("unable to add an item to the work queue, it's full: %d\n", len(p.queue))
			p.stalls++
		}
		time.Sleep(d)
	}
	//fmt.Printf("producer[%d]: exiting: competed %d, queue full %d times\n", tid, desiredWorkCount-remainingWork, fullCount)

	return nil
}

type producerFactory struct {
	lock    sync.Mutex
	created uint64
	stalls  uint64
}

func (pf *producerFactory) New(q workerpool.SubmissionQueue) (workerpool.Producer, error) {
	return &producer{queue: q}, nil
}

func (pf *producerFactory) Finished(producerIface workerpool.Producer) {
	p := producerIface.(*producer)

	pf.lock.Lock()
	defer pf.lock.Unlock()

	pf.created += p.created
	pf.stalls += p.stalls
}

type worker struct {
	queue  workerpool.SubmissionQueue
	done   uint64
	stalls uint64
}

func newWorker(q workerpool.SubmissionQueue) *worker {
	return &worker{queue: q}
}

func (w *worker) Run(ctx context.Context, tid workerpool.ThreadID) error {
CHANNEL_CLOSED:
	for {
		select {
		case t, ok := <-w.queue:
			_ = t
			if !ok {
				break CHANNEL_CLOSED
			}
			w.done++
		case <-ctx.Done():
			fmt.Printf("worker[%d]: shutting down\n", tid)
			break CHANNEL_CLOSED
		default:
			w.stalls++
		}

		d := 2 * time.Second
		//fmt.Printf("worker[%d]: Sleeping for %s\n", tid, d)
		time.Sleep(d)
		//fmt.Printf("worker[%d]: Done sleeping\n", tid)
	}
	//fmt.Printf("worker[%d]: exiting: completed %d, stalled %d times\n", tid, w.done, w.stalls)

	return nil
}

type workerFactory struct {
	lock   sync.Mutex
	done   uint64
	stalls uint64
}

func (wf *workerFactory) New(q workerpool.SubmissionQueue) (workerpool.Worker, error) {
	return &worker{queue: q}, nil
}

func (wf *workerFactory) Finished(workerIface workerpool.Worker) {
	w := workerIface.(*worker)

	wf.lock.Lock()
	defer wf.lock.Unlock()

	wf.done += w.done
	wf.stalls += w.stalls
}

func realMain() int {
	seed.MustInit()

	wf := workerFactory{}
	pf := producerFactory{}

	app := workerpool.New(
		workerpool.Config{
			InitialNumWorkers:   5,
			InitialNumProducers: 1,
			WorkQueueDepth:      2,
		},
		workerpool.Handlers{
			Reload: nil,
		},
		workerpool.Threads{
			ProducerFactory: &pf,
			WorkerFactory:   &wf,
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

	fmt.Printf("Work Created: %d\n", pf.created)
	fmt.Printf("Work Completed: %d\n", wf.done)
	fmt.Printf("Producer Stalls: %d\n", pf.stalls)
	fmt.Printf("Worker Stalls: %d\n", wf.stalls)

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
