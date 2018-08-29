package workerpool

// Config is the initial configuration of the workerpool
type Config struct {
	InitialNumProducers uint
	InitialNumConsumers uint
	WorkQueueDepth      uint
}
