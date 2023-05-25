package main

import (
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCliConfig_ValidateInitConfig(t *testing.T) {
	for _, testCase := range []struct {
		desc    string
		cfg     cliConfig
		wantErr error
	}{
		{
			desc: "missing config path",
			cfg: cliConfig{
				outputPath: "/foo/bar",
			},
			wantErr: errors.New("missing config flag"),
		},
		{
			desc: "missing output path",
			cfg: cliConfig{
				configPath: "/foo/bar",
			},
			wantErr: errors.New("missing output flag"),
		},
		{
			desc: "ok",
			cfg: cliConfig{
				configPath: "/foo/bar",
				outputPath: "/biz/buz",
			},
			wantErr: nil,
		},
	} {
		t.Run(testCase.desc, func(t *testing.T) {
			assert.Equal(
				t,
				testCase.wantErr,
				testCase.cfg.validateInitConfig(),
			)
		})
	}
}

func TestCliConfig_ValidateRuntimeConfig(t *testing.T) {
	hostname, err := os.Hostname()
	require.NoError(t, err)

	for _, testCase := range []struct {
		desc         string
		cfg          cliConfig
		wantMemberID string
		wantErr      error
	}{
		{
			desc: "ok",
			cfg: cliConfig{
				leaseName:              "lease",
				leaseNamespace:         "namespace",
				memberID:               "bloupi",
				notifyHTTPURL:          "http://reload.com",
				notifyHTTPMethod:       http.MethodPost,
				notifyRetryMaxAttempts: 1,
				notifyRetryDelay:       10 * time.Second,
				readinessPollPeriod:    10 * time.Second,
				apiListenAddr:          ":5678",
				apiShutdownGraceDelay:  15 * time.Second,
			},
			wantMemberID: "bloupi",
			wantErr:      nil,
		},
		{
			desc: "falls back to hostname if memberID is empty",
			cfg: cliConfig{
				leaseName:              "lease",
				leaseNamespace:         "namespace",
				memberID:               "",
				notifyHTTPURL:          "http://reload.com",
				notifyHTTPMethod:       http.MethodPost,
				notifyRetryMaxAttempts: 1,
				notifyRetryDelay:       10 * time.Second,
				readinessPollPeriod:    10 * time.Second,
				apiListenAddr:          ":5678",
				apiShutdownGraceDelay:  15 * time.Second,
			},
			wantMemberID: hostname,
			wantErr:      nil,
		},
		{
			desc: "missing leaseName",
			cfg: cliConfig{
				leaseName:              "",
				leaseNamespace:         "namespace",
				notifyHTTPURL:          "http://reload.com",
				notifyHTTPMethod:       http.MethodPost,
				notifyRetryMaxAttempts: 1,
				notifyRetryDelay:       10 * time.Second,
				readinessPollPeriod:    10 * time.Second,
				apiListenAddr:          ":5678",
				apiShutdownGraceDelay:  15 * time.Second,
			},
			wantErr: errors.New("missing lease-name flag"),
		},
		{
			desc: "missing lease namespace",
			cfg: cliConfig{
				leaseName:              "lease",
				leaseNamespace:         "",
				notifyHTTPURL:          "http://reload.com",
				notifyHTTPMethod:       http.MethodPost,
				notifyRetryMaxAttempts: 1,
				notifyRetryDelay:       10 * time.Second,
				readinessPollPeriod:    10 * time.Second,
				apiListenAddr:          ":5678",
				apiShutdownGraceDelay:  15 * time.Second,
			},
			wantErr: errors.New("missing lease-namespace flag"),
		},
		{
			desc: "missing lease notify-http-url",
			cfg: cliConfig{
				leaseName:              "lease",
				leaseNamespace:         "namespace",
				notifyHTTPURL:          "",
				notifyHTTPMethod:       http.MethodPost,
				notifyRetryMaxAttempts: 1,
				notifyRetryDelay:       10 * time.Second,
				readinessPollPeriod:    10 * time.Second,
				apiListenAddr:          ":5678",
				apiShutdownGraceDelay:  15 * time.Second,
			},
			wantErr: errors.New("missing notify-http-url flag"),
		},
		{
			desc: "missing lease notify-http-method",
			cfg: cliConfig{
				leaseName:              "lease",
				leaseNamespace:         "namespace",
				notifyHTTPURL:          "http://reload.com",
				notifyHTTPMethod:       "",
				notifyRetryMaxAttempts: 1,
				notifyRetryDelay:       10 * time.Second,
				readinessPollPeriod:    10 * time.Second,
				apiListenAddr:          ":5678",
				apiShutdownGraceDelay:  15 * time.Second,
			},
			wantErr: errors.New("invalid notify-http-method"),
		},
		{
			desc: "invalid lease notify-http-method",
			cfg: cliConfig{
				leaseName:              "lease",
				leaseNamespace:         "namespace",
				notifyHTTPURL:          "http://reload.com",
				notifyHTTPMethod:       "///3eee",
				notifyRetryMaxAttempts: 1,
				notifyRetryDelay:       10 * time.Second,
				readinessPollPeriod:    10 * time.Second,
				apiListenAddr:          ":5678",
				apiShutdownGraceDelay:  15 * time.Second,
			},
			wantErr: errors.New("invalid notify-http-method"),
		},
		{
			desc: "invalid retry-max-attempts",
			cfg: cliConfig{
				leaseName:              "lease",
				leaseNamespace:         "namespace",
				notifyHTTPURL:          "http://reload.com",
				notifyHTTPMethod:       http.MethodPost,
				notifyRetryMaxAttempts: -1,
				notifyRetryDelay:       10 * time.Second,
				readinessPollPeriod:    10 * time.Second,
				apiListenAddr:          ":5678",
				apiShutdownGraceDelay:  15 * time.Second,
			},
			wantErr: errors.New("invalid notify-retry-max-attempts, should be >= 1"),
		},
		{
			desc: "invalid retry-delay",
			cfg: cliConfig{
				leaseName:              "lease",
				leaseNamespace:         "namespace",
				notifyHTTPURL:          "http://reload.com",
				notifyHTTPMethod:       http.MethodPost,
				notifyRetryMaxAttempts: 1,
				notifyRetryDelay:       -10 * time.Second,
				readinessPollPeriod:    10 * time.Second,
				apiListenAddr:          ":5678",
				apiShutdownGraceDelay:  15 * time.Second,
			},
			wantErr: errors.New("invalid notify-retry-delay, should be >= 1"),
		},
		{
			desc: "missing api-listen-address",
			cfg: cliConfig{
				leaseName:              "lease",
				leaseNamespace:         "namespace",
				notifyHTTPURL:          "http://reload.com",
				notifyHTTPMethod:       http.MethodPost,
				notifyRetryMaxAttempts: 1,
				notifyRetryDelay:       10 * time.Second,
				readinessPollPeriod:    10 * time.Second,
				apiListenAddr:          "",
				apiShutdownGraceDelay:  15 * time.Second,
			},
			wantErr: errors.New("missing api-listen-address"),
		},
		{
			desc: "invalid api-shutdown-grace-delay",
			cfg: cliConfig{
				leaseName:              "lease",
				leaseNamespace:         "namespace",
				notifyHTTPURL:          "http://reload.com",
				notifyHTTPMethod:       http.MethodPost,
				notifyRetryMaxAttempts: 1,
				notifyRetryDelay:       10 * time.Second,
				readinessPollPeriod:    10 * time.Second,
				apiListenAddr:          ":5678",
				apiShutdownGraceDelay:  -15 * time.Second,
			},
			wantErr: errors.New("invalid api-shudown-grace-delay, should be >= 0"),
		},
		{
			desc: "invalid readiness poll period",
			cfg: cliConfig{
				leaseName:              "lease",
				leaseNamespace:         "namespace",
				notifyHTTPURL:          "http://reload.com",
				notifyHTTPMethod:       http.MethodPost,
				notifyRetryMaxAttempts: 1,
				notifyRetryDelay:       10 * time.Second,
				readinessPollPeriod:    -10 * time.Second,
				apiListenAddr:          ":5678",
				apiShutdownGraceDelay:  15 * time.Second,
			},
			wantErr: errors.New("invalid readiness-poll-period, should be >= 1"),
		},

		{
			desc: "proxy enabled invalid prometheus local port",
			cfg: cliConfig{
				leaseName:                     "lease",
				leaseNamespace:                "namespace",
				notifyHTTPURL:                 "http://reload.com",
				notifyHTTPMethod:              http.MethodPost,
				notifyRetryMaxAttempts:        1,
				notifyRetryDelay:              10 * time.Second,
				readinessPollPeriod:           10 * time.Second,
				apiListenAddr:                 ":5678",
				apiShutdownGraceDelay:         15 * time.Second,
				apiProxyEnabled:               true,
				apiProxyPrometheusLocalPort:   0,
				apiProxyPrometheusRemotePort:  9090,
				apiProxyPrometheusServiceName: "prometheus",
			},
			wantErr: errors.New("invalid api-proxy-prometheus-local-port, should be > 0"),
		},
		{
			desc: "proxy enabled invalid prometheus remote port",
			cfg: cliConfig{
				leaseName:                     "lease",
				leaseNamespace:                "namespace",
				notifyHTTPURL:                 "http://reload.com",
				notifyHTTPMethod:              http.MethodPost,
				notifyRetryMaxAttempts:        1,
				notifyRetryDelay:              10 * time.Second,
				readinessPollPeriod:           10 * time.Second,
				apiListenAddr:                 ":5678",
				apiShutdownGraceDelay:         15 * time.Second,
				apiProxyEnabled:               true,
				apiProxyPrometheusLocalPort:   9090,
				apiProxyPrometheusRemotePort:  0,
				apiProxyPrometheusServiceName: "prometheus",
			},
			wantErr: errors.New("invalid api-proxy-prometheus-remote-port, should be > 0"),
		},
		{
			desc: "proxy enabled missing prometheus service name",
			cfg: cliConfig{
				leaseName:                     "lease",
				leaseNamespace:                "namespace",
				notifyHTTPURL:                 "http://reload.com",
				notifyHTTPMethod:              http.MethodPost,
				notifyRetryMaxAttempts:        1,
				notifyRetryDelay:              10 * time.Second,
				readinessPollPeriod:           10 * time.Second,
				apiListenAddr:                 ":5678",
				apiShutdownGraceDelay:         15 * time.Second,
				apiProxyEnabled:               true,
				apiProxyPrometheusLocalPort:   9090,
				apiProxyPrometheusRemotePort:  9090,
				apiProxyPrometheusServiceName: "",
			},
			wantErr: errors.New("missing api-proxy-prometheus-service-name"),
		},
	} {
		t.Run(testCase.desc, func(t *testing.T) {
			assert.Equal(
				t,
				testCase.wantErr,
				testCase.cfg.validateRuntimeConfig(),
			)

			if testCase.wantMemberID != "" {
				assert.Equal(
					t,
					testCase.wantMemberID,
					testCase.cfg.memberID,
				)
			}

		})
	}
}
