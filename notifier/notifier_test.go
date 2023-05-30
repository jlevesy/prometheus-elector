package notifier_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jlevesy/prometheus-elector/notifier"
)

func TestHTTPNotifierWithRetries(t *testing.T) {
	var (
		totalReceived int
		reg           = prometheus.NewRegistry()
		ctx           = context.Background()
		srv           = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			require.Equal(t, r.Method, http.MethodPost)
			totalReceived++

			if totalReceived < 5 {
				rw.WriteHeader(http.StatusInternalServerError)
			}
		}))
		notifier = notifier.WithRetry(
			notifier.WithMetrics(
				reg,
				notifier.NewHTTP(
					srv.URL,
					http.MethodPost,
					time.Second,
				),
			),
			10,
			0*time.Second,
		)
	)

	const wantMetrics = `
# HELP prometheus_elector_notifier_calls_errors The total amount of times Prometheus Elector failed to notify Prometheus about a configuration update
# TYPE prometheus_elector_notifier_calls_errors counter
prometheus_elector_notifier_calls_errors 4
# HELP prometheus_elector_notifier_calls_total The total amount of times Prometheus Elector notified Prometheus about a configuration update
# TYPE prometheus_elector_notifier_calls_total counter
prometheus_elector_notifier_calls_total{} 5
`

	defer srv.Close()

	err := notifier.Notify(ctx)
	require.NoError(t, err)

	assert.Equal(t, 5, totalReceived)
	assert.NoError(t, testutil.GatherAndCompare(
		reg,
		bytes.NewBuffer([]byte(wantMetrics)),
		"prometheus_elector_notifier_calls_errors",
		"prometheus_elector_notifier_calls_total",
	))
}

func TestHTTPNotifierExhaustRetries(t *testing.T) {
	var (
		totalReceived int
		ctx           = context.Background()
		srv           = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			require.Equal(t, r.Method, http.MethodPost)
			totalReceived++
			rw.WriteHeader(http.StatusInternalServerError)
		}))
		notifier = notifier.WithRetry(
			notifier.NewHTTP(
				srv.URL,
				http.MethodPost,
				time.Second,
			),
			10,
			0*time.Second,
		)
	)

	defer srv.Close()

	err := notifier.Notify(ctx)
	require.ErrorContains(t, err, "notifier exhausted all retries")

	assert.Equal(t, 10, totalReceived)
}
