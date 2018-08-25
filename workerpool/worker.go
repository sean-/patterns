package workerpool

import "context"

type Worker interface {
	Run(context.Context, ThreadID) error
}
