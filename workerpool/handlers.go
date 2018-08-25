package workerpool

type Handlers struct {
	Reload                func()
	ProducerFactoryNewErr func(error)
	ProducerRunErr        func(error)
	WorkerRunErr          func(error)
	WorkerFactoryNewErr   func(error)
}

func (h Handlers) Copy() Handlers {
	return Handlers{
		Reload:                h.Reload,
		ProducerFactoryNewErr: h.ProducerFactoryNewErr,
		ProducerRunErr:        h.ProducerRunErr,
		WorkerRunErr:          h.WorkerRunErr,
		WorkerFactoryNewErr:   h.WorkerFactoryNewErr,
	}
}
