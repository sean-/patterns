package workerpool

type ProducerFactory interface {
	New(SubmissionQueue) (Producer, error)
}