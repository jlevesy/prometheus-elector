package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/jlevesy/prometheus-elector/config"
	"github.com/jlevesy/prometheus-elector/election"
	"github.com/jlevesy/prometheus-elector/notifier"
	"github.com/jlevesy/prometheus-elector/watcher"
)

func main() {
	var (
		leaseName          string
		leaseNamespace     string
		leaseDuration      time.Duration
		leaseRenewDeadline time.Duration
		leaseRetryPeriod   time.Duration
		kubeConfigPath     string
		configPath         string
		outputPath         string
		reloadURL          string
		init               bool

		err error

		memberID = os.Getenv("POD_NAME")
	)
	flag.StringVar(&leaseName, "lease-name", "", "Name of lease lock")
	flag.StringVar(&leaseNamespace, "lease-namespace", "", "Name of lease lock namespace")
	flag.DurationVar(&leaseDuration, "lease-duration", 15*time.Second, "Duration of a lease, client wait the full duration of a lease before trying to take it over")
	flag.DurationVar(&leaseRenewDeadline, "lease-renew-deadline", 10*time.Second, "Maximum duration spent trying to renew the lease")
	flag.DurationVar(&leaseRetryPeriod, "lease-retry-period", 2*time.Second, "Delay between two attempts of taking/renewing the lease")
	flag.StringVar(&kubeConfigPath, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&configPath, "config", "", "Path of the prometheus-elector configuration")
	flag.StringVar(&outputPath, "output", "", "Path to write the active prometheus configuration")
	flag.StringVar(&reloadURL, "reload-url", "", "URL to the reload configuration endpoint")
	flag.BoolVar(&init, "init", false, "Only init the prometheus config file")
	flag.Parse()

	if configPath == "" {
		klog.Fatal("missing config path")
	}
	if outputPath == "" {
		klog.Fatal("missing output path")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	reconciller := config.NewReconciller(configPath, outputPath)

	if err := reconciller.Reconcile(ctx); err != nil {
		klog.Fatal("Can't sync configuration: ", err)
	}

	if init {
		klog.Info("Running in init mode, exiting")
		return
	}

	if leaseName == "" {
		klog.Fatal("missing lease-name flag")
	}

	if leaseNamespace == "" {
		klog.Fatal("missing lease-namespace flag")
	}

	if reloadURL == "" {
		klog.Fatal("missing reloadURL path")
	}

	if memberID == "" {
		memberID, err = os.Hostname()
		if err != nil {
			klog.Fatal("can't read hostname: ", err)
		}
	}

	notifier := notifier.NewHTTP(reloadURL, http.MethodPost)

	k8sConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		klog.Fatal("Unable to build kube client configuration: ", err)
	}

	k8sClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		klog.Fatal("Can't build the k8s client: ", err)
	}

	watcher, err := watcher.New(filepath.Dir(configPath), reconciller, notifier)
	if err != nil {
		klog.Fatal("Can't create the watcher: ", err)
	}
	defer watcher.Close()

	elector, err := election.Setup(
		election.Config{
			LeaseName:       leaseName,
			LeaseNamespace:  leaseNamespace,
			ReleaseOnCancel: true,
			LeaseDuration:   leaseDuration,
			RenewDeadline:   leaseRenewDeadline,
			RetryPeriod:     leaseRetryPeriod,
			MemberID:        memberID,
		},
		k8sClient,
		reconciller,
		notifier,
	)

	if err != nil {
		klog.Fatal("Can't setup election", err)
	}

	grp, grpCtx := errgroup.WithContext(ctx)

	grp.Go(func() error {
		elector.Run(grpCtx)
		return nil
	})

	grp.Go(func() error {
		return watcher.Watch(grpCtx)
	})

	if err := grp.Wait(); err != nil {
		klog.Fatal("leader-agent failed, reason: ", err)
	}

	klog.Info("Leader-Agent exited successfully")
}
