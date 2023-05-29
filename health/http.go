package health

import (
	"context"
	"errors"
	"net/http"
	"time"

	"k8s.io/klog/v2"
)

type HTTPCheckConfig struct {
	URL              string
	Period           time.Duration
	Timeout          time.Duration
	SuccessThreshold int
	FailureThreshold int
}

type HTTPChecker struct {
	config    HTTPCheckConfig
	callbacks Callbacks

	httpClient *http.Client
}

func NewHTTPChecker(cfg HTTPCheckConfig, cbs Callbacks) *HTTPChecker {
	return &HTTPChecker{
		callbacks:  cbs,
		config:     cfg,
		httpClient: http.DefaultClient,
	}
}

func (c *HTTPChecker) Check(ctx context.Context) error {
	klog.InfoS("Starting healtcheck", "url", c.config.URL, "period", c.config.Period)
	ticker := time.NewTicker(c.config.Period)
	defer ticker.Stop()

	var status checkState

	for {
		select {
		case <-ticker.C:
			ok, err := c.doCheck(ctx)
			if errors.Is(err, context.Canceled) {
				return nil
			}

			if err != nil {
				klog.ErrorS(err, "unable to perform health check, exiting")
				return err
			}

			if ok {
				status.successCount++
				status.failureCount = 0
			} else {
				status.failureCount++
				status.successCount = 0
			}

			if status.successCount == c.config.SuccessThreshold {
				if err := c.callbacks.OnHealthy(); err != nil {
					klog.ErrorS(err, "Unable to notify healthiness")
				}
			}

			if status.failureCount == c.config.FailureThreshold {
				if err := c.callbacks.OnUnHealthy(); err != nil {
					klog.ErrorS(err, "Unable to notify unhealthiness")
				}
			}

		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				return nil
			}

			return err
		}

	}
}

func (c *HTTPChecker) doCheck(ctx context.Context) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.config.URL, http.NoBody)
	if err != nil {
		return false, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		klog.ErrorS(err, "unable to query the health endpoint")
		return false, nil
	}

	// [200, 300[ then OK.
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return true, nil
	}

	klog.InfoS("Prometheus failed an healthcheck", "code", resp.StatusCode)

	return false, nil
}

type checkState struct {
	successCount int
	failureCount int
}
