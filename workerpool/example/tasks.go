package main

import (
	"errors"
	"math/rand"
	"time"

	"github.com/rs/zerolog"
	"github.com/sean-/patterns/workerpool"
)

type taskReal struct {
	log zerolog.Logger

	tid      workerpool.ThreadID
	finishFn func(error)
}

func (rt *taskReal) DoWork() error {
	time.Sleep(2 * time.Second)

	if rand.Intn(5) == 1 {
		return errors.New("randomly broke real task")
	}
	return nil
}

type taskCanary struct {
	log zerolog.Logger

	tid      workerpool.ThreadID
	finishFn func(error)
}

func (ct *taskCanary) DoWork() error {
	time.Sleep(10 * time.Millisecond)

	if rand.Intn(5) < 2 {
		return errors.New("randomly broke canary task")
	}
	return nil
}
