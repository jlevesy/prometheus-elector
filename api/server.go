package api

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"
)

type Server struct {
	httpSrv http.Server

	shutdownGraceDelay time.Duration
}

func NewServer(listenAddr string, shutdownGraceDelay time.Duration, metricsRegistry prometheus.Gatherer) *Server {
	var mux http.ServeMux

	mux.HandleFunc("/healthz", func(rw http.ResponseWriter, r *http.Request) { rw.WriteHeader(http.StatusOK) })
	mux.Handle("/metrics", promhttp.HandlerFor(
		metricsRegistry,
		promhttp.HandlerOpts{EnableOpenMetrics: true},
	))

	return &Server{
		shutdownGraceDelay: shutdownGraceDelay,
		httpSrv: http.Server{
			Addr:    listenAddr,
			Handler: &mux,
		},
	}
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
