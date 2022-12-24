package notifier_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jlevesy/prometheus-elector/notifier"
)

func TestHTTPNotifierWithRetries(t *testing.T) {
	var (
		totalReceived int
		ctx           = context.Background()
		srv           = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			require.Equal(t, r.Method, http.MethodPost)
			totalReceived++

			if totalReceived < 5 {
				rw.WriteHeader(http.StatusInternalServerError)
			}
		}))
		notifier = notifier.WithRetry(
			notifier.NewHTTP(
				srv.URL,
				http.MethodPost,
			),
			10,
			0*time.Second,
		)
	)

	defer srv.Close()

	err := notifier.Notify(ctx)
	require.NoError(t, err)

	assert.Equal(t, 5, totalReceived)
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
