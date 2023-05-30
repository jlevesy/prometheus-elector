package health_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jlevesy/prometheus-elector/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPChecker(t *testing.T) {
	for _, tt := range []struct {
		desc     string
		sequence []int

		successThreshold      int
		failureThreshold      int
		wantOnHealthyCalled   int
		wantOnUnhealthyCalled int
	}{
		{
			desc: "calls healty",
			sequence: []int{
				http.StatusOK,
				http.StatusOK,
				http.StatusOK,
				http.StatusOK,
			},
			wantOnHealthyCalled: 1,
		},
		{
			desc: "calls unhealthy",
			sequence: []int{
				http.StatusInternalServerError,
				http.StatusInternalServerError,
				http.StatusInternalServerError,
				http.StatusInternalServerError,
			},
			wantOnUnhealthyCalled: 1,
		},
		{
			desc: "resets state",
			sequence: []int{
				http.StatusOK,
				http.StatusInternalServerError,
				http.StatusOK,
				http.StatusOK,
				http.StatusInternalServerError,
				http.StatusInternalServerError,
				http.StatusOK,
				http.StatusOK,
				http.StatusOK,
				http.StatusOK,
				http.StatusInternalServerError,
				http.StatusInternalServerError,
				http.StatusInternalServerError,
				http.StatusInternalServerError,
			},
			wantOnUnhealthyCalled: 1,
			wantOnHealthyCalled:   1,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			var (
				callCount    int
				sequenceDone = make(chan struct{})
				srv          = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
					if callCount == len(tt.sequence)-1 {
						close(sequenceDone)
						rw.WriteHeader(http.StatusOK)
						return
					}

					rw.WriteHeader(tt.sequence[callCount])
					callCount++
				}))
			)
			defer srv.Close()

			var (
				onHealthyCalled   int
				onUnhealthyCalled int

				callbacks = health.CallbacksFuncs{
					OnHealthyFunc: func() error {
						onHealthyCalled++
						return nil
					},
					OnUnHealthyFunc: func() error {
						onUnhealthyCalled++
						return nil
					},
				}

				config = health.HTTPCheckConfig{
					URL:              srv.URL + "/-/health",
					Period:           time.Millisecond,
					Timeout:          time.Second,
					SuccessThreshold: 3,
					FailureThreshold: 3,
				}

				checkDone = make(chan struct{})
				checker   = health.NewHTTPChecker(config, callbacks)
			)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				err := checker.Check(ctx)
				require.NoError(t, err)
				close(checkDone)
			}()

			<-sequenceDone
			cancel()
			<-checkDone

			assert.Equal(t, tt.wantOnHealthyCalled, onHealthyCalled)
			assert.Equal(t, tt.wantOnUnhealthyCalled, onUnhealthyCalled)
		})
	}
}
