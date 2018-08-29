package main

import (
	"sync"

	"github.com/sean-/patterns/workerpool"
)

type consumerFactory struct {
	lock                sync.Mutex
	completed           uint64
	stalls              uint64
	workCompletedCanary uint64
	workCompletedReal   uint64
}

func (wf *consumerFactory) New(q workerpool.SubmissionQueue) (workerpool.Consumer, error) {
	return &consumer{queue: q}, nil
}

func (wf *consumerFactory) Finished(threadID workerpool.ThreadID, consumerIface workerpool.Consumer) {
	w := consumerIface.(*consumer)

	wf.lock.Lock()
	defer wf.lock.Unlock()
	wf.workCompletedCanary += w.workCompletedCanary
	wf.completed += w.completed
	wf.workCompletedReal += w.workCompletedReal
}
