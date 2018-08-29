package workerpool

type ConsumerFactory interface {
	New(SubmissionQueue) (Consumer, error)
	Finished(ThreadID, Consumer)
}
