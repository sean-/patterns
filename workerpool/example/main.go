package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sean-/patterns/workerpool"
	"github.com/sean-/seed"
	"github.com/sean-/sysexits"
)

// Interface is the workerpool interface that is exported
type Interface interface {
	// ShutdownCtx is the shutdown context, used to poll for the status of shutdown
	ShutdownCtx() context.Context

	// InitiateShutdown cancels the context used to signal its time to shut down
	InitiateShutdown() (bool, error)

	// Reload is a configuration reload handler (e.g. for use with SIGHUP)
	Reload()
}

func main() {
	os.Exit(realMain())
}

func realMain() int {
	seed.MustInit()

	wf := &consumerFactory{}
	pf := &producerFactory{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := workerpool.New(
		workerpool.Config{
			InitialNumConsumers: 5,
			InitialNumProducers: 1,
			WorkQueueDepth:      10,
		},
		workerpool.Factories{
			ProducerFactory: pf,
			ConsumerFactory: wf,
		},
		workerpool.Handlers{
			ReloadFunc:   nil,
			ShutdownCtx:  ctx,
			ShutdownFunc: cancel,
		},
	)

	if err := runSignalHandler(app); err != nil {
		fmt.Printf("unable to launch signal handler: %v", err)
		return sysexits.Software
	}

	if err := app.StartProducers(); err != nil {
		fmt.Printf("unable to start the workerpool producers: %v", err)
		return sysexits.Software
	}
	fmt.Println("started producers")

	if err := app.StartConsumers(); err != nil {
		fmt.Printf("unable to start the workerpool consumers: %v", err)
		return sysexits.Software
	}
	fmt.Println("started consumers")

	if err := app.WaitProducers(); err != nil {
		fmt.Printf("error waiting for workpool producers: %v", err)
		return sysexits.Software
	}
	fmt.Println("finished waiting for producers")

	if err := app.WaitConsumers(); err != nil {
		fmt.Printf("error waiting for the workerpool to drain: %v", err)
		return sysexits.Software
	}
	fmt.Println("finished waiting for consumers")

	fmt.Printf("Work Submitted: %d\n", pf.submittedWork)
	fmt.Printf("Work Completed: %d\n", wf.completed)
	fmt.Printf("  Real Work Completed:   %d\n", wf.workCompletedReal)
	fmt.Printf("  Canary Work Completed: %d\n", wf.workCompletedCanary)
	fmt.Printf("Producer Stalls: %d\n", pf.stalls)
	fmt.Printf("Consumer Stalls: %d\n", wf.stalls)

	return sysexits.OK
}

func runSignalHandler(wp Interface) error {
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
