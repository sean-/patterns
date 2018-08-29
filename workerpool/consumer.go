package workerpool

import "context"

type Consumer interface {
	Run(context.Context, ThreadID) error
}
