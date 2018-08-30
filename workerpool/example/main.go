package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const logTimeFormat = "2006-01-02T15:04:05.000000000Z07:00"

	zerolog.DurationFieldUnit = time.Microsecond
	zerolog.DurationFieldInteger = true
	zerolog.TimeFieldFormat = logTimeFormat

	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
		With().Timestamp().Logger()

	pf := NewProducerFactory(log)
	cf := NewConsumerFactory(log)

	app := workerpool.New(
		workerpool.Config{
			InitialNumConsumers: 5,
			InitialNumProducers: 1,
			WorkQueueDepth:      10,
		},
		workerpool.Factories{
			ProducerFactory: pf,
			ConsumerFactory: cf,
		},
		workerpool.Handlers{
			ReloadFunc:   nil,
			ShutdownCtx:  ctx,
			ShutdownFunc: cancel,
		},
	)

	if err := runSignalHandler(log, app); err != nil {
		log.Error().Err(err).Msg("unable to launch signal handler")
		return sysexits.Software
	}

	if err := app.StartProducers(); err != nil {
		log.Error().Err(err).Msg("unable to start the workerpool producers")
		return sysexits.Software
	}
	log.Info().Msg("started producers")

	if err := app.StartConsumers(); err != nil {
		log.Error().Err(err).Msg("unable to start the workerpool consumers")
		return sysexits.Software
	}
	log.Info().Msg("started consumers")

	if err := app.WaitProducers(); err != nil {
		log.Error().Err(err).Msg("error waiting for workpool producers")
		return sysexits.Software
	}
	log.Info().Msg("finished waiting for producers")

	if err := app.WaitConsumers(); err != nil {
		log.Error().Err(err).Msg("error waiting for the workerpool to drain")
		return sysexits.Software
	}
	log.Info().Msg("finished waiting for consumers")

	log.Info().Msgf("Work Submitted: %d", pf.submittedWork)
	log.Info().Msgf("Work Completed: %d", cf.completed)
	log.Info().Msgf("\tReal Work Completed:   %d", cf.workCompletedReal)
	log.Info().Msgf("\tCanary Work Completed: %d", cf.workCompletedCanary)
	log.Info().Msgf("Producer Stalls: %d", pf.stalls)
	log.Info().Msgf("Consumer Stalls: %d", cf.stalls)

	return sysexits.OK
}

func runSignalHandler(log zerolog.Logger, wp Interface) error {
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
				log.Info().Msg("shutdown initiated, terminating signal handler")
				wp.InitiateShutdown()
				return
			case sig := <-sigCh:
				switch sig {
				case syscall.SIGHUP:
					log.Info().Msgf("received %s, reloading", sig)
					wp.Reload()
				case syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM:
					log.Info().Msgf("received %s, shutting down", sig)
					wp.InitiateShutdown()
					return
				}
			}
		}
	}()

	return nil
}
