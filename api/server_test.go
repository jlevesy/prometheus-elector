package api_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/jlevesy/prometheus-elector/api"
)

func TestServer_Serve(t *testing.T) {
	var (
		ctx, cancel = context.WithCancel(context.Background())

		srv = api.NewServer(
			":63549",
			15*time.Second,
			prometheus.NewRegistry(),
		)

		srvDone = make(chan struct{})
	)

	go func() {
		err := srv.Serve(ctx)
		require.NoError(t, err)

		close(srvDone)
	}()

	var attempt = 0

	for {
		time.Sleep(200 * time.Millisecond)
		attempt += 1

		if attempt == 5 {
			t.Fatal("Exausted max retries getting the /healthz endpoint")
		}

		resp, err := http.Get("http://localhost:63549/healthz")
		if err != nil {
			continue
		}
		if resp.StatusCode != http.StatusOK {
			continue
		}

		break
	}

	cancel()
	<-srvDone
}
