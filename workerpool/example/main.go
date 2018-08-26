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

type producer struct {
	queue         workerpool.SubmissionQueue
	submittedWork uint64
	stalls        uint64

	// pacingDuration is the delay to introduce when we're able to flood work into
	// the queue
	pacingDuration time.Duration

	// backoffDuration is the delay when the max in-flight operations has been
	// exceeded
	backoffDuration time.Duration

	taskRealLock  sync.Mutex
	maxRealTasks  uint64
	currRealTasks uint64

	taskCanaryLock  sync.Mutex
	maxCanaryTasks  uint64
	currCanaryTasks uint64
}

func (p *producer) Run(ctx context.Context, tid workerpool.ThreadID) error {
	const desiredWorkCount = 100
	remainingWork := desiredWorkCount

STOP_PRODUCING_WORK:
	for remainingWork > 0 {
		var task workerpool.Task

		p.taskRealLock.Lock()
		if p.currRealTasks < p.maxRealTasks {
			p.currRealTasks++
			p.taskRealLock.Unlock()
			taskID := remainingWork
			_ = taskID
			rt := &taskReal{
				finishFn: func() {
					p.taskRealLock.Lock()
					defer p.taskRealLock.Unlock()
					p.currRealTasks--
				},
			}
			task = rt
		} else {
			p.taskRealLock.Unlock()
		}

		if task == nil {
			p.taskCanaryLock.Lock()
			if p.currCanaryTasks < p.maxCanaryTasks {
				p.currCanaryTasks++
				p.taskCanaryLock.Unlock()
				taskID := remainingWork
				_ = taskID
				ct := &taskCanary{
					finishFn: func() {
						p.taskCanaryLock.Lock()
						defer p.taskCanaryLock.Unlock()
						p.currCanaryTasks--
					},
				}
				task = ct
			} else {
				p.taskCanaryLock.Unlock()
			}
		}

		if task == nil {
			//fmt.Printf("producer[%d]: backing off, too much work in-flight\n", tid)
			time.Sleep(p.backoffDuration)
			continue
		}

		select {
		case p.queue <- task:
			p.submittedWork++
			remainingWork--
			//fmt.Printf("producer[%d]: added a work item: %d remaining\n", tid, remainingWork)
		case <-ctx.Done():
			//fmt.Printf("producer[%d]: shutting down\n", tid)
			break STOP_PRODUCING_WORK
		default:
			fmt.Printf("unable to add an item to the work queue, it's full: %d\n", len(p.queue))
			p.stalls++
		}
		time.Sleep(p.pacingDuration)
	}
	//fmt.Printf("producer[%d]: exiting: competed %d, stalled %d times\n", tid, desiredWorkCount-remainingWork, p.stalls)

	return nil
}

type producerFactory struct {
	lock          sync.Mutex
	submittedWork uint64
	stalls        uint64
}

func (pf *producerFactory) New(q workerpool.SubmissionQueue) (workerpool.Producer, error) {
	p := &producer{
		queue:           q,
		pacingDuration:  100 * time.Millisecond,
		backoffDuration: 1 * time.Second,
		maxRealTasks:    3,
		maxCanaryTasks:  10,
	}

	return p, nil
}

func (pf *producerFactory) Finished(threadID workerpool.ThreadID, producerIface workerpool.Producer) {
	p := producerIface.(*producer)

	pf.lock.Lock()
	defer pf.lock.Unlock()

	pf.submittedWork += p.submittedWork
	pf.stalls += p.stalls
}

type worker struct {
	queue               workerpool.SubmissionQueue
	completed           uint64
	workCompletedReal   uint64
	workCompletedCanary uint64
}

func newWorker(q workerpool.SubmissionQueue) *worker {
	return &worker{queue: q}
}

func (w *worker) Run(ctx context.Context, tid workerpool.ThreadID) error {
CHANNEL_CLOSED:
	for {
		select {
		case t, ok := <-w.queue:
			if !ok {
				break CHANNEL_CLOSED
			}
			w.completed++

			switch task := t.(type) {
			case *taskReal:
				task.DoWork()
				task.finishFn()
				w.workCompletedReal++
			case *taskCanary:
				task.DoWork()
				task.finishFn()
				w.workCompletedCanary++
			default:
				panic(fmt.Sprintf("invalid type: %v", task))
			}

		case <-ctx.Done():
			fmt.Printf("worker[%d]: shutting down\n", tid)
			break CHANNEL_CLOSED
		}
	}
	//fmt.Printf("worker[%d]: exiting: completed %d, stalled %d times\n", tid, w.completed, w.stalls)

	return nil
}

type workerFactory struct {
	lock                sync.Mutex
	completed           uint64
	stalls              uint64
	workCompletedCanary uint64
	workCompletedReal   uint64
}

func (wf *workerFactory) New(q workerpool.SubmissionQueue) (workerpool.Worker, error) {
	return &worker{queue: q}, nil
}

func (wf *workerFactory) Finished(threadID workerpool.ThreadID, workerIface workerpool.Worker) {
	w := workerIface.(*worker)

	wf.lock.Lock()
	defer wf.lock.Unlock()
	wf.workCompletedCanary += w.workCompletedCanary
	wf.completed += w.completed
	wf.workCompletedReal += w.workCompletedReal
}

func realMain() int {
	seed.MustInit()

	wf := workerFactory{}
	pf := producerFactory{}

	app := workerpool.New(
		workerpool.Config{
			InitialNumWorkers:   5,
			InitialNumProducers: 1,
			WorkQueueDepth:      10,
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

	fmt.Printf("Work Submitted: %d\n", pf.submittedWork)
	fmt.Printf("Work Completed: %d\n", wf.completed)
	fmt.Printf("  Real Work Completed:   %d\n", wf.workCompletedReal)
	fmt.Printf("  Canary Work Completed: %d\n", wf.workCompletedCanary)
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
