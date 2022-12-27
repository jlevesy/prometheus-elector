package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jlevesy/prometheus-elector/api"
)

func TestServer_ServeHTTP_ProxyNotLeaderForwardsToLeader(t *testing.T) {
	var (
		ctx, cancel  = context.WithCancel(context.Background())
		srvDone      = make(chan struct{})
		callReceived int
	)

	defer cancel()

	backend := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "localhost:63549", r.Host)
		assert.Equal(t, "/some/path", r.URL.Path)
		callReceived += 1
	}))

	defer backend.Close()

	url, err := url.Parse(backend.URL)
	require.NoError(t, err)

	portInt, err := strconv.ParseUint(url.Port(), 10, 64)
	require.NoError(t, err)

	srv, err := api.NewServer(
		api.Config{
			ListenAddress:         ":63549",
			ShutdownGraceDelay:    15 * time.Second,
			EnableLeaderProxy:     true,
			PrometheusRemotePort:  uint(portInt),
			PrometheusServiceName: "localhost",
		},
		&leaderStatusStub{
			isLeader: false,
			leader:   "bozo",
		},
		prometheus.NewRegistry(),
	)
	require.NoError(t, err)

	go func() {
		err := srv.Serve(ctx)
		require.NoError(t, err)

		close(srvDone)
	}()

	require.NoError(t, waitForServerReady(5))

	resp, err := http.Get("http://localhost:63549/some/path")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = http.Get("http://localhost:63549/some/path")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assert.Equal(t, 2, callReceived)

	cancel()
	<-srvDone
}

func TestServer_ServeHTTP_ProxyIsLeaderForwardsToLocalhost(t *testing.T) {
	var (
		ctx, cancel  = context.WithCancel(context.Background())
		srvDone      = make(chan struct{})
		callReceived int
	)

	defer cancel()

	backend := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "localhost:63549", r.Host)
		assert.Equal(t, "/some/path", r.URL.Path)
		callReceived += 1
	}))

	defer backend.Close()

	url, err := url.Parse(backend.URL)
	require.NoError(t, err)

	portInt, err := strconv.ParseUint(url.Port(), 10, 64)
	require.NoError(t, err)

	srv, err := api.NewServer(
		api.Config{
			ListenAddress:       ":63549",
			ShutdownGraceDelay:  15 * time.Second,
			EnableLeaderProxy:   true,
			PrometheusLocalPort: uint(portInt),
		},
		&leaderStatusStub{
			isLeader: true,
			leader:   "bozo",
		},
		prometheus.NewRegistry(),
	)
	require.NoError(t, err)

	go func() {
		err := srv.Serve(ctx)
		require.NoError(t, err)

		close(srvDone)
	}()

	require.NoError(t, waitForServerReady(5))

	resp, err := http.Get("http://localhost:63549/some/path")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assert.Equal(t, 1, callReceived)

	cancel()
	<-srvDone
}

func TestServer_ServeHTTP_ProxyDisabled(t *testing.T) {
	var (
		ctx, cancel = context.WithCancel(context.Background())
		srvDone     = make(chan struct{})
	)

	defer cancel()

	srv, err := api.NewServer(
		api.Config{
			ListenAddress:      ":63549",
			ShutdownGraceDelay: 15 * time.Second,
		},
		&leaderStatusStub{
			isLeader: true,
			leader:   "bozo",
		},
		prometheus.NewRegistry(),
	)
	require.NoError(t, err)

	go func() {
		err := srv.Serve(ctx)
		require.NoError(t, err)

		close(srvDone)
	}()

	require.NoError(t, waitForServerReady(5))

	resp, err := http.Get("http://localhost:63549/_elector/metrics")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = http.Get("http://localhost:63549/_elector/leader")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	defer resp.Body.Close()

	var gotLeaderStatus api.LeaderStatus

	err = json.NewDecoder(resp.Body).Decode(&gotLeaderStatus)
	require.NoError(t, err)
	assert.Equal(
		t,
		api.LeaderStatus{
			IsLeader:      true,
			CurrentLeader: "bozo",
		},
		gotLeaderStatus,
	)

	resp, err = http.Get("http://localhost:63549/api/v1/range_query")
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	cancel()
	<-srvDone
}

func waitForServerReady(maxAttempts int) error {
	var attempt int

	for {
		time.Sleep(200 * time.Millisecond)
		attempt += 1

		if attempt == maxAttempts {
			return errors.New("exausted max retries getting the /_elector/healthz endpoint")
		}

		resp, err := http.Get("http://localhost:63549/_elector/healthz")
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			continue
		}

		break
	}

	return nil
}

type leaderStatusStub struct {
	leader   string
	isLeader bool
}

func (s *leaderStatusStub) IsLeader() bool    { return s.isLeader }
func (s *leaderStatusStub) GetLeader() string { return s.leader }
