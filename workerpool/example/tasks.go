package main

import "time"

type taskReal struct {
	finishFn func()
}

func (rt *taskReal) DoWork() {
	time.Sleep(2 * time.Second)
}

type taskCanary struct {
	finishFn func()
}

func (ct *taskCanary) DoWork() {
	time.Sleep(10 * time.Millisecond)
}
