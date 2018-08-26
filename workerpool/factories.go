package workerpool

// ThreadID is a generic identifier for a given thread
type ThreadID int

// Factories is a container for different types of factories
type Factories struct {
	ProducerFactory ProducerFactory
	WorkerFactory   WorkerFactory
}
