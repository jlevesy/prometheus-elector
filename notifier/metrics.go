package notifier

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metricsNotifier struct {
	next Notifier

	total    prometheus.Counter
	errors   prometheus.Counter
	duration prometheus.Histogram
}

func WithMetrics(reg prometheus.Registerer, notifier Notifier) Notifier {
	return &metricsNotifier{
		next: notifier,

		total: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Namespace: "prometheus_elector",
			Name:      "notifier_calls_total",
			Help:      "The total amount of times Prometheus Elector notified Prometheus about a configuration update",
		}),
		errors: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Namespace: "prometheus_elector",
			Name:      "notifier_calls_errors",
			Help:      "The total amount of times Prometheus Elector failed to notify Prometheus about a configuration update",
		}),
		duration: promauto.With(reg).NewHistogram(prometheus.HistogramOpts{
			Namespace: "prometheus_elector",
			Name:      "notifier_calls_duration_seconds",
			Help:      "The time it took to notify prometheus about a configuration update",
		}),
	}
}

func (m *metricsNotifier) Notify(ctx context.Context) error {
	return m.intrument(func() error {
		return m.next.Notify(ctx)
	})
}

func (m *metricsNotifier) intrument(cb func() error) error {
	startTime := time.Now()

	err := cb()

	if err != nil {
		m.errors.Inc()
	}

	m.duration.Observe(time.Since(startTime).Seconds())
	m.total.Inc()
	return err
}
