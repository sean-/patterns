package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sean-/patterns/workerpool"
	"github.com/sean-/seed"
	"github.com/sean-/sysexits"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	seed.MustInit()

	wf := &workerFactory{}
	pf := &producerFactory{}

	app := workerpool.New(
		workerpool.Config{
			InitialNumWorkers:   5,
			InitialNumProducers: 1,
			WorkQueueDepth:      10,
		},
		workerpool.Factories{
			ProducerFactory: pf,
			WorkerFactory:   wf,
		},
		workerpool.Handlers{
			Reload: nil,
		},
	)
	defer app.InitiateShutdown()

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
