package workerpool

type WorkerFactory interface {
	New(SubmissionQueue) (Worker, error)
	Finished(ThreadID, Worker)
}
