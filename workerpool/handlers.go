package workerpool

type Handlers struct {
	Reload                func()
	ProducerFactoryNewErr func(error)
	ProducerRunErr        func(error) (resumable bool)
	WorkerRunErr          func(error) (resumable bool)
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
