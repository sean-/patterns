package workerpool

type Config struct {
	InitialNumProducers uint
	InitialNumWorkers   uint
	WorkQueueDepth      uint
}

func (cfg Config) Copy() Config {
	return Config{
		InitialNumProducers: cfg.InitialNumProducers,
		InitialNumWorkers:   cfg.InitialNumWorkers,
		WorkQueueDepth:      cfg.WorkQueueDepth,
	}
}
