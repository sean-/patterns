package main

import (
	"sync"
	"time"

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

// rampFactor is a multiplier used to calculate the duration of initial ramp up
const rampFactor = 5

func NewConsumerFactory(log zerolog.Logger) *consumerFactory {
	return &consumerFactory{
		log: log,
	}
}

func (cf *consumerFactory) New(tid workerpool.ThreadID, q workerpool.SubmissionQueue) (workerpool.Consumer, error) {
	return &consumer{
		log:          cf.log,
		tid:          tid,
		queue:        q,
		rampDuration: time.Duration(tid*rampFactor) * time.Second,
	}, nil
}

func (cf *consumerFactory) Finished(tid workerpool.ThreadID, consumerIface workerpool.Consumer) {
	w := consumerIface.(*consumer)

	cf.lock.Lock()
	defer cf.lock.Unlock()

	cf.workCompletedCanary += w.workCompletedCanary
	cf.completed += w.completed
	cf.workCompletedReal += w.workCompletedReal
}
