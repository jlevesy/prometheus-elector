package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/klog/v2"

	"github.com/jlevesy/prometheus-elector/api"
	"github.com/jlevesy/prometheus-elector/config"
	"github.com/jlevesy/prometheus-elector/election"
	"github.com/jlevesy/prometheus-elector/health"
	"github.com/jlevesy/prometheus-elector/notifier"
	"github.com/jlevesy/prometheus-elector/readiness"
	"github.com/jlevesy/prometheus-elector/watcher"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		cfg         = newCLIConfig()
		ctx, cancel = signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)

		promReady   = make(chan struct{})
		grp, grpCtx = errgroup.WithContext(ctx)
	)
	defer cancel()
	cfg.setupFlags()

	flag.Parse()

	if err := cfg.validateInitConfig(); err != nil {
		klog.ErrorS(err, "Invalid init config")
		return 1
	}

	reconciller := config.NewReconciller(cfg.configPath, cfg.outputPath)

	if err := reconciller.Reconcile(ctx); err != nil {
		klog.ErrorS(err, "Can't perform an initial sync")
		return 1
	}

	if cfg.init {
		klog.Info("Running in init mode, exiting")
		return 0
	}

	if err := cfg.validateRuntimeConfig(); err != nil {
		klog.ErrorS(err, "Invalid election config")
		return 1
	}

	metricsRegistry := prometheus.NewRegistry()
	if cfg.runtimeMetrics {
		metricsRegistry.MustRegister(collectors.NewBuildInfoCollector())
		metricsRegistry.MustRegister(collectors.NewGoCollector(
			collectors.WithGoCollectorRuntimeMetrics(collectors.MetricsAll),
		))
	}

	notifier := notifier.WithRetry(
		notifier.WithMetrics(
			metricsRegistry,
			notifier.NewHTTP(
				cfg.notifyHTTPURL,
				cfg.notifyHTTPMethod,
				cfg.notifyTimeout,
			),
		),
		cfg.notifyRetryMaxAttempts,
		cfg.notifyRetryDelay,
	)

	k8sConfig, err := clientcmd.BuildConfigFromFlags("", cfg.kubeConfigPath)
	if err != nil {
		klog.ErrorS(err, "Unable to build kube client configuration")
		return 1
	}

	k8sClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		klog.ErrorS(err, "Can't build the Kubernetes client")
		return 1
	}

	elector, err := election.New(
		election.Config{
			LeaseName:      cfg.leaseName,
			LeaseNamespace: cfg.leaseNamespace,
			LeaseDuration:  cfg.leaseDuration,
			RenewDeadline:  cfg.leaseRenewDeadline,
			RetryPeriod:    cfg.leaseRetryPeriod,
			MemberID:       cfg.memberID,
		},
		k8sClient,
		leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				klog.Info("Leading, applying leader configuration.")

				if err := reconciller.Reconcile(ctx); err != nil {
					klog.ErrorS(err, "Failed to reconcile configurations")
					return
				}

				if err := notifier.Notify(ctx); err != nil {
					klog.ErrorS(err, "Failed to notify prometheus")
					return
				}
			},
			OnStoppedLeading: func() {
				klog.Info("Stopped leading, applying follower configuration.")

				if err := reconciller.Reconcile(ctx); err != nil {
					klog.ErrorS(err, "Failed to reconcile configurations")
					return
				}

				if err := notifier.Notify(ctx); err != nil {
					klog.ErrorS(err, "Failed to notify prometheus")
					return
				}
			},
		},
		metricsRegistry,
	)
	if err != nil {
		klog.ErrorS(err, "Can't setup the election")
		return 1
	}

	// Always leave the election.
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		klog.Info("Graceful shutdown, leaving the election")

		if err := elector.Stop(stopCtx); err != nil && errors.Is(err, election.ErrNotRunning) {
			klog.ErrorS(err, "Unable to leave the election")
		}

		klog.Info("Graceful shutdown, left the election")
	}()

	reconciller.SetLeaderChecker(elector.Status())

	watcher, err := watcher.New(filepath.Dir(cfg.configPath), reconciller, notifier)
	if err != nil {
		klog.ErrorS(err, "Can't create the watcher")
		return 1
	}
	defer watcher.Close()

	apiServer, err := api.NewServer(
		api.Config{
			ListenAddress:         cfg.apiListenAddr,
			ShutdownGraceDelay:    cfg.apiShutdownGraceDelay,
			EnableLeaderProxy:     cfg.apiProxyEnabled,
			PrometheusLocalPort:   cfg.apiProxyPrometheusLocalPort,
			PrometheusRemotePort:  cfg.apiProxyPrometheusRemotePort,
			PrometheusServiceName: cfg.apiProxyPrometheusServiceName,
		},
		elector.Status(),
		metricsRegistry,
	)

	if err != nil {
		klog.ErrorS(err, "Can't set up the API server")
		return 1
	}

	var readinessWaiter readiness.Waiter = readiness.NoopWaiter{}

	if cfg.readinessHTTPURL != "" {
		readinessWaiter = readiness.NewHTTP(
			cfg.readinessHTTPURL,
			cfg.readinessPollPeriod,
			cfg.readinessTimeout,
		)
	}

	var healthChecker health.Checker = health.NoopChecker{}

	if cfg.healthcheckHTTPURL != "" {
		healthChecker = health.NewHTTPChecker(
			health.HTTPCheckConfig{
				URL:              cfg.healthcheckHTTPURL,
				Period:           cfg.healthcheckPeriod,
				Timeout:          cfg.healthcheckTimeout,
				SuccessThreshold: cfg.healthcheckSuccessThreshold,
				FailureThreshold: cfg.healthcheckFailureThreshold,
			},
			health.CallbacksFuncs{
				OnHealthyFunc: func() error {
					klog.Info("Prometheus is healthy, joining the election")
					err := elector.Start(grpCtx)
					if errors.Is(err, election.ErrAlreadyRunning) {
						klog.Info("Already joined the election, ignoring.")
						return nil
					}

					return err
				},
				OnUnHealthyFunc: func() error {
					klog.Info("Prometheus is unhealthy, leaving the election")
					err := elector.Stop(grpCtx)
					if errors.Is(err, election.ErrNotRunning) {
						klog.Info("Already left the election, ignoring.")
						return nil
					}

					return err
				},
			},
		)

	}

	grp.Go(func() error {
		if err := readinessWaiter.Wait(grpCtx); err != nil {
			return err
		}

		// Signal whoever is waiting that prometheus is ready.
		close(promReady)

		// Join election as soon as we start.
		err := elector.Start(grpCtx)
		if errors.Is(err, election.ErrAlreadyRunning) {
			return nil
		}

		return err
	})

	grp.Go(func() error {
		// Wait for the app to be ready before starting the healthcheck.
		select {
		case <-grpCtx.Done():
			return nil
		case <-promReady:
		}

		return healthChecker.Check(grpCtx)
	})

	grp.Go(func() error { return watcher.Watch(grpCtx) })
	grp.Go(func() error { return apiServer.Serve(grpCtx) })

	if err := grp.Wait(); err != nil {
		klog.ErrorS(err, "prometheus-elector has reported an error while running")
		return 1
	}

	klog.Info("prometheus-elector is gracefully stopping")
	return 0
}
