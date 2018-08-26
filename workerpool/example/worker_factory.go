package main

import (
	"sync"

	"github.com/sean-/patterns/workerpool"
)

type workerFactory struct {
	lock                sync.Mutex
	completed           uint64
	stalls              uint64
	workCompletedCanary uint64
	workCompletedReal   uint64
}

func (wf *workerFactory) New(q workerpool.SubmissionQueue) (workerpool.Worker, error) {
	return &worker{queue: q}, nil
}

func (wf *workerFactory) Finished(threadID workerpool.ThreadID, workerIface workerpool.Worker) {
	w := workerIface.(*worker)

	wf.lock.Lock()
	defer wf.lock.Unlock()
	wf.workCompletedCanary += w.workCompletedCanary
	wf.completed += w.completed
	wf.workCompletedReal += w.workCompletedReal
}
