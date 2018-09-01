package main

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/sean-/patterns/workerpool"
)

type producerFactory struct {
	log zerolog.Logger

	lock          sync.Mutex
	submittedWork uint64
	stalls        uint64
}

func NewProducerFactory(log zerolog.Logger) *producerFactory {
	return &producerFactory{
		log: log,
	}
}

func (pf *producerFactory) New(tid workerpool.ThreadID, q workerpool.SubmissionQueue) (workerpool.Producer, error) {
	const maxRealTasks = 10
	p := &producer{
		log:             pf.log,
		tid:             tid,
		queue:           q,
		pacingDuration:  1 * time.Millisecond,
		backoffDuration: 100 * time.Millisecond,
		maxRealTasks:    maxRealTasks,
		maxCanaryTasks:  1000 - maxRealTasks,
	}

	return p, nil
}

func (pf *producerFactory) Finished(tid workerpool.ThreadID, producerIface workerpool.Producer) {
	p := producerIface.(*producer)

	pf.lock.Lock()
	defer pf.lock.Unlock()

	pf.submittedWork += p.submittedWork
	pf.stalls += p.stalls
}
