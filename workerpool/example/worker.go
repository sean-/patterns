package main

import (
	"context"
	"fmt"

	"github.com/sean-/patterns/workerpool"
)

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
EXIT:
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("worker[%d]: shutting down\n", tid)
			break EXIT
		case t, ok := <-w.queue:
			if !ok {
				break EXIT // channel closed
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
		}
	}
	//fmt.Printf("worker[%d]: exiting: completed %d, stalled %d times\n", tid, w.completed, w.stalls)

	return nil
}
