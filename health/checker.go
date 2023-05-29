package health

import (
	"context"
)

type CallbacksFuncs struct {
	OnHealthyFunc   func() error
	OnUnHealthyFunc func() error
}

func (c CallbacksFuncs) OnHealthy() error {
	if c.OnHealthyFunc == nil {
		return nil
	}

	return c.OnHealthyFunc()
}

func (c CallbacksFuncs) OnUnHealthy() error {
	if c.OnUnHealthyFunc == nil {
		return nil
	}

	return c.OnUnHealthyFunc()
}

type Callbacks interface {
	OnHealthy() error
	OnUnHealthy() error
}

type Checker interface {
	Check(ctx context.Context) error
}

type NoopChecker struct{}

func (n NoopChecker) Check(context.Context) error {
	return nil
}
