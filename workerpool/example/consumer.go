package main

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/sean-/patterns/workerpool"
)

type consumer struct {
	log zerolog.Logger

	queue               workerpool.SubmissionQueue
	completed           uint64
	workCompletedReal   uint64
	workCompletedCanary uint64
}

func (c *consumer) Run(ctx context.Context, tid workerpool.ThreadID) error {
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
			c.completed++

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
		}
	}
	c.log.Debug().Msgf("consumer[%d]: exiting: completed %d times\n", tid, c.completed)

	return nil
}
