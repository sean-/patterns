package workerpool

type Threads struct {
	ProducerFactory ProducerFactory
	WorkerFactory   WorkerFactory
}

type ThreadID int

func (t Threads) Copy() Threads {
	return Threads{
		ProducerFactory: t.ProducerFactory,
		WorkerFactory:   t.WorkerFactory,
	}
}
