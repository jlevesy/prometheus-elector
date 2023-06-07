package notifier

import (
	"context"
	"errors"
	"fmt"
	"time"

	"k8s.io/klog/v2"
)

type retryNotifier struct {
	next        Notifier
	delay       time.Duration
	maxAttempts int
}

func WithRetry(next Notifier, maxAttempts int, delay time.Duration) Notifier {
	return &retryNotifier{
		next:        next,
		delay:       delay,
		maxAttempts: maxAttempts,
	}
}

func (r *retryNotifier) Notify(ctx context.Context) error {
	var err error

	for j := r.maxAttempts; j > 0; j-- {
		if err = r.next.Notify(ctx); err == nil {
			return nil
		}

		if errors.Is(err, context.Canceled) {
			return nil
		}

		if j > 0 {
			klog.ErrorS(err, "Failed to notify prometheus, will retry...", "attempt", r.maxAttempts-j, "maxAttempts", r.maxAttempts)
			time.Sleep(r.delay)
		}
	}

	return fmt.Errorf("notifier exhausted all retries: %w", err)
}
