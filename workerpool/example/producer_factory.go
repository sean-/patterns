package main

import (
	"sync"
	"time"

	"github.com/sean-/patterns/workerpool"
)

type producerFactory struct {
	lock          sync.Mutex
	submittedWork uint64
	stalls        uint64
}

func (pf *producerFactory) New(q workerpool.SubmissionQueue) (workerpool.Producer, error) {
	p := &producer{
		queue:           q,
		pacingDuration:  100 * time.Millisecond,
		backoffDuration: 1 * time.Second,
		maxRealTasks:    3,
		maxCanaryTasks:  10,
	}

	return p, nil
}

func (pf *producerFactory) Finished(threadID workerpool.ThreadID, producerIface workerpool.Producer) {
	p := producerIface.(*producer)

	pf.lock.Lock()
	defer pf.lock.Unlock()

	pf.submittedWork += p.submittedWork
	pf.stalls += p.stalls
}
