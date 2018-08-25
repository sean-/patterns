package workerpool

import "context"

type Producer interface {
	Run(context.Context, ThreadID) error
}
