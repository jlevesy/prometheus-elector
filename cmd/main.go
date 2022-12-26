package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"

	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/jlevesy/prometheus-elector/api"
	"github.com/jlevesy/prometheus-elector/config"
	"github.com/jlevesy/prometheus-elector/election"
	"github.com/jlevesy/prometheus-elector/notifier"
	"github.com/jlevesy/prometheus-elector/watcher"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

func main() {
	var (
		cfg         = newCLIConfig()
		ctx, cancel = signal.NotifyContext(context.Background(), os.Interrupt)
	)

	defer cancel()
	cfg.setupFlags()

	flag.Parse()

	if err := cfg.validateInitConfig(); err != nil {
		klog.Fatal("Invalid init config: ", err)
	}

	reconciller := config.NewReconciller(cfg.configPath, cfg.outputPath)

	if err := reconciller.Reconcile(ctx); err != nil {
		klog.Fatal("Can't perform an initial sync: ", err)
	}

	if cfg.init {
		klog.Info("Running in init mode, exiting")
		return
	}

	if err := cfg.validateRuntimeConfig(); err != nil {
		klog.Fatal("Invalid election config: ", err)
	}

	metricsRegistry := prometheus.NewRegistry()
	metricsRegistry.MustRegister(collectors.NewBuildInfoCollector())
	metricsRegistry.MustRegister(collectors.NewGoCollector(
		collectors.WithGoCollectorRuntimeMetrics(collectors.MetricsAll),
	))

	notifier := notifier.WithRetry(
		notifier.WithMetrics(
			metricsRegistry,
			notifier.NewHTTP(
				cfg.notifyHTTPURL,
				cfg.notifyHTTPMethod,
			),
		),
		cfg.notifyRetryMaxAttempts,
		cfg.notifyRetryDelay,
	)

	k8sConfig, err := clientcmd.BuildConfigFromFlags("", cfg.kubeConfigPath)
	if err != nil {
		klog.Fatal("Unable to build kube client configuration: ", err)
	}

	k8sClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		klog.Fatal("Can't build the k8s client: ", err)
	}

	watcher, err := watcher.New(filepath.Dir(cfg.configPath), reconciller, notifier)
	if err != nil {
		klog.Fatal("Can't create the watcher: ", err)
	}
	defer watcher.Close()

	elector, err := election.Setup(
		election.Config{
			LeaseName:       cfg.leaseName,
			LeaseNamespace:  cfg.leaseNamespace,
			ReleaseOnCancel: true,
			LeaseDuration:   cfg.leaseDuration,
			RenewDeadline:   cfg.leaseRenewDeadline,
			RetryPeriod:     cfg.leaseRetryPeriod,
			MemberID:        cfg.memberID,
		},
		k8sClient,
		reconciller,
		notifier,
		metricsRegistry,
	)

	if err != nil {
		klog.Fatal("Can't setup election", err)
	}

	apiServer := api.NewServer(
		cfg.apiListenAddr,
		cfg.apiShutdownGraceDelay,
		metricsRegistry,
	)

	grp, grpCtx := errgroup.WithContext(ctx)

	grp.Go(func() error {
		elector.Run(grpCtx)
		return nil
	})

	grp.Go(func() error { return watcher.Watch(grpCtx) })
	grp.Go(func() error { return apiServer.Serve(grpCtx) })

	if err := grp.Wait(); err != nil {
		klog.Fatal("leader-agent failed, reason: ", err)
	}

	klog.Info("Leader-Agent exited successfully")
}
