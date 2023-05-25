package readiness

import "context"

type Waiter interface {
	Wait(ctx context.Context) error
}

type NoopWaiter struct{}

func (NoopWaiter) Wait(context.Context) error {
	return nil
}
