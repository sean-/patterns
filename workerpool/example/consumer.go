package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/sean-/patterns/workerpool"
)

type consumer struct {
	log zerolog.Logger

	tid                 workerpool.ThreadID
	queue               workerpool.SubmissionQueue
	completed           uint64
	workCompletedReal   uint64
	workCompletedCanary uint64
	rampDuration        time.Duration
}

// rampUp linearly in increasing increments of the 5 seconds. This function is
// only called once when each consumer is first Run.
func (c *consumer) rampUp() {
	c.log.Debug().Msgf("consumer[%d] waiting %s to start", c.tid, c.rampDuration)
	time.Sleep(c.rampDuration)
}

// Run runs the consumer for each work task submitted by the producer.
func (c *consumer) Run(ctx context.Context) error {
	var once sync.Once

EXIT:
	for {
		select {
		case <-ctx.Done():
			c.log.Debug().Msgf("consumer[%d]: shutting down", c.tid)
			break EXIT
		case t, ok := <-c.queue:
			if !ok {
				break EXIT // channel closed
			}

			// NOTE: this isn't just an example of ramp up but also any prestart
			// function that only needs to run once initially for each consumer
			once.Do(c.rampUp)

			switch task := t.(type) {
			case *taskReal:
				c.log.Debug().Msgf("consumer[%d]: received real work task", c.tid)

				task.DoWork()
				task.finishFn()
				c.workCompletedReal++
			case *taskCanary:
				c.log.Debug().Msgf("consumer[%d]: received canary work task", c.tid)

				task.DoWork()
				task.finishFn()
				c.workCompletedCanary++
			default:
				panic(fmt.Sprintf("invalid type: %v", task))
			}
			c.completed++
		}
	}
	c.log.Debug().Msgf("consumer[%d]: exiting: completed %d times\n", c.tid, c.completed)

	return nil
}
