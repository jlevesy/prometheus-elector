package readiness_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jlevesy/prometheus-elector/readiness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPWaiter(t *testing.T) {
	var checkCalled bool

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/foo", r.URL.Path)

		if checkCalled {
			rw.WriteHeader(http.StatusOK)
			return
		}

		checkCalled = true
		rw.WriteHeader(http.StatusInsufficientStorage)
	}))
	defer srv.Close()

	waiter := readiness.NewHTTP(srv.URL+"/foo", 200*time.Millisecond)

	err := waiter.Wait(context.Background())
	require.NoError(t, err)
	assert.True(t, checkCalled)
}
