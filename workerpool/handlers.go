package workerpool

type Handlers struct {
	Reload                func()
	ProducerFactoryNewErr func(error)
	ProducerRunErr        func(error) (resumable bool)
	ConsumerRunErr        func(error) (resumable bool)
	ConsumerFactoryNewErr func(error)
}
