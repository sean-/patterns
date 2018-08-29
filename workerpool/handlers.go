package workerpool

import "context"

type Handlers struct {
	ReloadFunc   func()
	ShutdownFunc func()
	ShutdownCtx  context.Context

	ProducerFactoryNewErr func(error)
	ConsumerFactoryNewErr func(error)

	ProducerRunErr func(error) (resumable bool)
	ConsumerRunErr func(error) (resumable bool)
}
