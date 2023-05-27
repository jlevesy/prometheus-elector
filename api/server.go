package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jlevesy/prometheus-elector/election"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"
)

type Server struct {
	httpSrv http.Server

	shutdownGraceDelay time.Duration
}

type Config struct {
	// HTTP Server configuration.
	ListenAddress      string
	ShutdownGraceDelay time.Duration

	// Proxy Configuration.
	EnableLeaderProxy     bool
	PrometheusLocalPort   uint
	PrometheusRemotePort  uint
	PrometheusServiceName string
}

type LeaderStatus struct {
	IsLeader      bool   `json:"is_leader"`
	CurrentLeader string `json:"current_leader"`
}

func NewServer(cfg Config, electionStatus election.Status, metricsRegistry prometheus.Gatherer) (*Server, error) {
	var mux http.ServeMux

	mux.HandleFunc("/_elector/leader", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)

		_ = json.NewEncoder(rw).Encode(LeaderStatus{
			IsLeader:      electionStatus.IsLeader(),
			CurrentLeader: electionStatus.GetLeader(),
		})
	})
	mux.HandleFunc("/_elector/healthz", func(rw http.ResponseWriter, r *http.Request) { rw.WriteHeader(http.StatusOK) })
	mux.Handle("/_elector/metrics", promhttp.HandlerFor(
		metricsRegistry,
		promhttp.HandlerOpts{EnableOpenMetrics: true},
	))

	if cfg.EnableLeaderProxy {
		leaderProxy, err := newProxy(cfg, electionStatus)
		if err != nil {
			return nil, err
		}

		mux.Handle("/", leaderProxy)
	}

	return &Server{
		shutdownGraceDelay: cfg.ShutdownGraceDelay,
		httpSrv: http.Server{
			Addr:    cfg.ListenAddress,
			Handler: &mux,
		},
	}, nil
}

func (s *Server) Serve(ctx context.Context) error {
	shutdownDone := make(chan error)

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownGraceDelay)
		defer cancel()

		err := s.httpSrv.Shutdown(shutdownCtx)
		if err != nil {
			klog.Info("Server shutdown reported an error, forcing close")
			err = s.httpSrv.Close()
		}

		shutdownDone <- err
	}()

	if err := s.httpSrv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return <-shutdownDone
}
