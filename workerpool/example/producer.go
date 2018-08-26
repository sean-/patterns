package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sean-/patterns/workerpool"
)

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
