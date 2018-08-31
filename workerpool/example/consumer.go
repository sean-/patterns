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
}

// rampUp linearly in increasing increments of the 5 seconds. This function is
// only called once when each consumer is first Run.
func (c *consumer) rampUp() {
	rampUpDur := time.Duration((c.tid)*5) * time.Second
	c.log.Debug().Msgf("consumer[%d] waiting %s to start", c.tid, rampUpDur)
	time.Sleep(rampUpDur)
}

// Run runs the consumer for each work task submitted by the producer.
func (c *consumer) Run(ctx context.Context, tid workerpool.ThreadID) error {
	var once sync.Once
	c.tid = tid

EXIT:
	for {
		select {
		case <-ctx.Done():
			c.log.Debug().Msgf("consumer[%d]: shutting down", tid)
			break EXIT
		case t, ok := <-c.queue:
			if !ok {
				break EXIT // channel closed
			}
			once.Do(c.rampUp)

			switch task := t.(type) {
			case *taskReal:
				c.log.Debug().Msgf("consumer[%d]: received real work task", tid)

				task.DoWork()
				task.finishFn()
				c.workCompletedReal++
			case *taskCanary:
				c.log.Debug().Msgf("consumer[%d]: received canary work task", tid)

				task.DoWork()
				task.finishFn()
				c.workCompletedCanary++
			default:
				panic(fmt.Sprintf("invalid type: %v", task))
			}
			c.completed++
		}
	}
	c.log.Debug().Msgf("consumer[%d]: exiting: completed %d times\n", tid, c.completed)

	return nil
}
