package main

import (
	"sync"

	"github.com/rs/zerolog"
	"github.com/sean-/patterns/workerpool"
)

type consumerFactory struct {
	log zerolog.Logger

	lock                sync.Mutex
	completed           uint64
	stalls              uint64
	workCompletedCanary uint64
	workCompletedReal   uint64
}

func NewConsumerFactory(log zerolog.Logger) *consumerFactory {
	return &consumerFactory{
		log: log,
	}
}

func (cf *consumerFactory) New(q workerpool.SubmissionQueue) (workerpool.Consumer, error) {
	return &consumer{
		log:   cf.log,
		queue: q,
	}, nil
}

func (cf *consumerFactory) Finished(threadID workerpool.ThreadID, consumerIface workerpool.Consumer) {
	w := consumerIface.(*consumer)

	cf.lock.Lock()
	defer cf.lock.Unlock()
	cf.workCompletedCanary += w.workCompletedCanary
	cf.completed += w.completed
	cf.workCompletedReal += w.workCompletedReal
}
