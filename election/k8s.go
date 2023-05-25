package election

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/klog/v2"

	"github.com/jlevesy/prometheus-elector/config"
	"github.com/jlevesy/prometheus-elector/notifier"
	"github.com/prometheus/client_golang/prometheus"
)

type Config struct {
	LeaseName       string
	LeaseNamespace  string
	MemberID        string
	ReleaseOnCancel bool
	LeaseDuration   time.Duration
	RenewDeadline   time.Duration
	RetryPeriod     time.Duration
}

func Setup(cfg Config, k8sClient kubernetes.Interface, reconciller *config.Reconciler, notifier notifier.Notifier, reg prometheus.Registerer) (*leaderelection.LeaderElector, error) {
	leaderelection.SetProvider(metricsProvider(func() leaderelection.SwitchMetric {
		return newLeaderMetrics(reg)
	}))

	le, err := leaderelection.NewLeaderElector(
		leaderelection.LeaderElectionConfig{
			Lock: &resourcelock.LeaseLock{
				LeaseMeta: metav1.ObjectMeta{
					Name:      cfg.LeaseName,
					Namespace: cfg.LeaseNamespace,
				},
				Client: k8sClient.CoordinationV1(),
				LockConfig: resourcelock.ResourceLockConfig{
					Identity: cfg.MemberID,
				},
			},
			Name:            cfg.MemberID, // required to properly set election metrics.
			ReleaseOnCancel: cfg.ReleaseOnCancel,
			LeaseDuration:   cfg.LeaseDuration,
			RenewDeadline:   cfg.RenewDeadline,
			RetryPeriod:     cfg.RetryPeriod,
			Callbacks: leaderelection.LeaderCallbacks{
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

					ctx := context.Background()

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
		},
	)

	if err != nil {
		return nil, err
	}

	reconciller.SetLeaderChecker(le)

	return le, nil

}
