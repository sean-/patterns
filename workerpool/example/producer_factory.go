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
	const maxRealTasks = 10
	p := &producer{
		queue:           q,
		pacingDuration:  1 * time.Millisecond,
		backoffDuration: 100 * time.Millisecond,
		maxRealTasks:    maxRealTasks,
		maxCanaryTasks:  1000 - maxRealTasks,
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
