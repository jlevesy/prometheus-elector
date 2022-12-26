package election

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/client-go/tools/leaderelection"
)

type metricsProvider func() leaderelection.SwitchMetric

func (mp metricsProvider) NewLeaderMetric() leaderelection.SwitchMetric {
	return mp()
}

type leaderMetrics struct {
	isLeader            *prometheus.GaugeVec
	lastTranstitionTime prometheus.Gauge
}

func newLeaderMetrics(r prometheus.Registerer) *leaderMetrics {
	return &leaderMetrics{
		lastTranstitionTime: promauto.With(r).NewGauge(
			prometheus.GaugeOpts{
				Namespace: "prometheus_elector",
				Name:      "election_last_transition_time_seconds",
				Help:      "last time the member changed status",
			},
		),
		isLeader: promauto.With(r).NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "prometheus_elector",
				Name:      "election_is_leader",
				Help:      "Set which member is the actual leader of the cluster",
			},
			[]string{"member_id"},
		),
	}
}

func (m *leaderMetrics) On(name string) {
	m.isLeader.WithLabelValues(name).Set(1.0)
	m.lastTranstitionTime.SetToCurrentTime()
}

func (m *leaderMetrics) Off(name string) {
	m.isLeader.WithLabelValues(name).Set(0.0)
	m.lastTranstitionTime.SetToCurrentTime()
}
