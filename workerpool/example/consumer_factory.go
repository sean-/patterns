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
	stalls              uint64
	completed           uint64
	errored             uint64
	workRequested       uint64
	workCompletedReal   uint64
	workCompletedCanary uint64
	workErrReal         uint64
	workErrCanary       uint64
}

// rampFactor is a multiplier used to calculate the duration of initial ramp up
const rampFactor = 5

// rampUnit is the increment of time used to calculate the duration of initial
// ramp up
const rampUnit = 1 * time.Second

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
		rampDuration: time.Duration(tid*rampFactor) * rampUnit,
	}, nil
}

func (cf *consumerFactory) Finished(tid workerpool.ThreadID, consumerIface workerpool.Consumer) {
	w := consumerIface.(*consumer)

	cf.lock.Lock()
	defer cf.lock.Unlock()

	cf.workCompletedCanary += w.workCompletedCanary
	cf.workCompletedReal += w.workCompletedReal
	cf.workErrCanary += w.workErrCanary
	cf.workErrReal += w.workErrReal

	cf.workRequested += w.workRequested
	cf.completed += w.completed
	cf.errored += w.workErrCanary + w.workErrReal
}
