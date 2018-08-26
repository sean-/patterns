package workerpool

type Handlers struct {
	Reload                func()
	ProducerFactoryNewErr func(error)
	ProducerRunErr        func(error) (resumable bool)
	WorkerRunErr          func(error) (resumable bool)
	WorkerFactoryNewErr   func(error)
}
