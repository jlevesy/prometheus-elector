package readiness

import (
	"context"
	"net/http"
	"time"

	"k8s.io/klog/v2"
)

type httpWaiter struct {
	url        string
	pollPeriod time.Duration

	httpClient *http.Client
}

func NewHTTP(url string, pollPeriod time.Duration) Waiter {
	return &httpWaiter{
		url:        url,
		pollPeriod: pollPeriod,
		httpClient: http.DefaultClient,
	}
}

func (w *httpWaiter) Wait(ctx context.Context) error {
	klog.InfoS("Waiting for prometheus to be ready", "poll_period", w.pollPeriod, "url", w.url)

	tick := time.NewTicker(w.pollPeriod)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			ready, err := w.checkReadiness(ctx)
			if err != nil {
				return err
			}

			if ready {
				klog.Info("Prometheus is ready")
				return nil
			}
		}
	}
}

func (w *httpWaiter) checkReadiness(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.url, http.NoBody)
	if err != nil {
		return false, err
	}

	rsp, err := w.httpClient.Do(req)
	if err != nil {
		klog.ErrorS(err, "Failed to check if Prometheus is ready")
		return false, nil
	}

	if rsp.StatusCode != http.StatusOK {
		klog.Error("Prometheus isn't ready yet", "status", rsp.StatusCode)
		return false, nil
	}

	return true, nil
}
