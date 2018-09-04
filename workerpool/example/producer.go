package main

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/sean-/patterns/workerpool"
)

type producer struct {
	log zerolog.Logger

	tid           workerpool.ThreadID
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

func (p *producer) Run(ctx context.Context) error {
EXIT:
	for {
		var task workerpool.Task

		p.taskRealLock.Lock()
		if p.currRealTasks < p.maxRealTasks {
			p.currRealTasks++
			p.taskRealLock.Unlock()

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
			p.log.Debug().Msgf("producer[%d]: backing off, too much work in-flight", p.tid)
			time.Sleep(p.backoffDuration)
			continue
		}

		select {
		case <-ctx.Done():
			p.log.Info().Msgf("producer[%d]: shutting down", p.tid)
			break EXIT
		case p.queue <- task:
			p.submittedWork++
			p.log.Debug().Msgf("producer[%d]: added a work item", p.tid)
			time.Sleep(p.pacingDuration)
		default:
			p.stalls++
			// Make a blocking write now that we've recorded the stall
			blockedAt := time.Now()
			select {
			case <-ctx.Done():
				break EXIT
			case p.queue <- task:
				p.submittedWork++
				// Smooth out the pacing
				stallDuration := time.Now().Sub(blockedAt)
				if stallDuration < p.pacingDuration {
					time.Sleep(p.pacingDuration - stallDuration)
				}
			}
		}
	}
	p.log.Debug().Msgf("producer[%d]: exiting: submitted %d, stalled %d times", p.tid, p.submittedWork, p.stalls)

	return nil
}
