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

var goodConfig = cliConfig{
	leaseName:                   "lease",
	leaseNamespace:              "namespace",
	memberID:                    "bloupi",
	notifyHTTPURL:               "http://reload.com",
	notifyHTTPMethod:            http.MethodPost,
	notifyRetryMaxAttempts:      1,
	notifyRetryDelay:            10 * time.Second,
	notifyTimeout:               time.Second,
	readinessPollPeriod:         10 * time.Second,
	readinessTimeout:            time.Second,
	healthcheckPeriod:           10 * time.Second,
	healthcheckTimeout:          10 * time.Second,
	healthcheckSuccessThreshold: 3,
	healthcheckFailureThreshold: 3,
	apiListenAddr:               ":5678",
	apiShutdownGraceDelay:       15 * time.Second,
}

var goodConfigWithProxy = cliConfig{
	leaseName:                     "lease",
	leaseNamespace:                "namespace",
	memberID:                      "bloupi",
	notifyHTTPURL:                 "http://reload.com",
	notifyHTTPMethod:              http.MethodPost,
	notifyRetryMaxAttempts:        1,
	notifyRetryDelay:              10 * time.Second,
	notifyTimeout:                 time.Second,
	readinessPollPeriod:           10 * time.Second,
	readinessTimeout:              time.Second,
	healthcheckPeriod:             10 * time.Second,
	healthcheckTimeout:            10 * time.Second,
	healthcheckSuccessThreshold:   3,
	healthcheckFailureThreshold:   3,
	apiListenAddr:                 ":5678",
	apiShutdownGraceDelay:         15 * time.Second,
	apiProxyEnabled:               true,
	apiProxyPrometheusLocalPort:   9095,
	apiProxyPrometheusRemotePort:  9090,
	apiProxyPrometheusServiceName: "prometheus",
}

func TestCliConfig_ValidateRuntimeConfig(t *testing.T) {
	hostname, err := os.Hostname()
	require.NoError(t, err)

	for _, testCase := range []struct {
		desc         string
		baseConfig   cliConfig
		patchConfig  func(c *cliConfig)
		wantMemberID string
		wantErr      error
	}{
		{
			desc:         "ok",
			baseConfig:   goodConfig,
			patchConfig:  func(c *cliConfig) {},
			wantMemberID: "bloupi",
			wantErr:      nil,
		},
		{
			desc:       "falls back to hostname if memberID is empty",
			baseConfig: goodConfig,
			patchConfig: func(c *cliConfig) {
				c.memberID = ""
			},
			wantMemberID: hostname,
			wantErr:      nil,
		},
		{
			desc:       "missing leaseName",
			baseConfig: goodConfig,
			patchConfig: func(c *cliConfig) {
				c.leaseName = ""
			},
			wantErr: errors.New("missing lease-name flag"),
		},
		{
			desc:       "missing lease namespace",
			baseConfig: goodConfig,
			patchConfig: func(c *cliConfig) {
				c.leaseNamespace = ""
			},
			wantErr: errors.New("missing lease-namespace flag"),
		},
		{
			desc:       "missing lease notify-http-url",
			baseConfig: goodConfig,
			patchConfig: func(c *cliConfig) {
				c.notifyHTTPURL = ""
			},
			wantErr: errors.New("missing notify-http-url flag"),
		},
		{
			desc:       "missing lease notify-http-method",
			baseConfig: goodConfig,
			patchConfig: func(c *cliConfig) {
				c.notifyHTTPMethod = ""
			},
			wantErr: errors.New("invalid notify-http-method"),
		},
		{
			desc:       "invalid notify http-method",
			baseConfig: goodConfig,
			patchConfig: func(c *cliConfig) {
				c.notifyHTTPMethod = "///3eee"
			},
			wantErr: errors.New("invalid notify-http-method"),
		},
		{
			desc:       "invalid notify retry-max-attempts",
			baseConfig: goodConfig,
			patchConfig: func(c *cliConfig) {
				c.notifyRetryMaxAttempts = -1
			},
			wantErr: errors.New("invalid notify-retry-max-attempts, should be >= 1"),
		},
		{
			desc:       "invalid notify retry-delay",
			baseConfig: goodConfig,
			patchConfig: func(c *cliConfig) {
				c.notifyRetryDelay = -10 * time.Second
			},
			wantErr: errors.New("invalid notify-retry-delay, should be >= 1"),
		},
		{
			desc:       "invalid notify timeout",
			baseConfig: goodConfig,
			patchConfig: func(c *cliConfig) {
				c.notifyTimeout = -10 * time.Second
			},
			wantErr: errors.New("invalid notify-timeout, should be >= 1"),
		},
		{
			desc:       "invalid healthcheck period",
			baseConfig: goodConfig,
			patchConfig: func(c *cliConfig) {
				c.healthcheckPeriod = -10 * time.Second
			},
			wantErr: errors.New("invalid healthcheck-period, should be >= 1"),
		},
		{
			desc:       "invalid healthcheck timeout",
			baseConfig: goodConfig,
			patchConfig: func(c *cliConfig) {
				c.healthcheckTimeout = -10 * time.Second
			},
			wantErr: errors.New("invalid healthcheck-timeout, should be >= 1"),
		},
		{
			desc:       "invalid healthcheck success threshold",
			baseConfig: goodConfig,
			patchConfig: func(c *cliConfig) {
				c.healthcheckSuccessThreshold = 0
			},
			wantErr: errors.New("invalid healthcheck-success-threshold, should be >= 1"),
		},
		{
			desc:       "invalid healthcheck failure threshold",
			baseConfig: goodConfig,
			patchConfig: func(c *cliConfig) {
				c.healthcheckFailureThreshold = 0
			},
			wantErr: errors.New("invalid healthcheck-failure-threshold, should be >= 1"),
		},
		{
			desc:       "missing api-listen-address",
			baseConfig: goodConfigWithProxy,
			patchConfig: func(c *cliConfig) {
				c.apiListenAddr = ""
			},
			wantErr: errors.New("missing api-listen-address"),
		},
		{
			desc:       "invalid api-shutdown-grace-delay",
			baseConfig: goodConfig,
			patchConfig: func(c *cliConfig) {
				c.apiShutdownGraceDelay = -15 * time.Second
			},
			wantErr: errors.New("invalid api-shudown-grace-delay, should be >= 0"),
		},
		{
			desc:       "invalid readiness poll period",
			baseConfig: goodConfig,
			patchConfig: func(c *cliConfig) {
				c.readinessPollPeriod = -10 * time.Second
			},
			wantErr: errors.New("invalid readiness-poll-period, should be >= 1"),
		},
		{
			desc:       "invalid readiness timeout",
			baseConfig: goodConfig,
			patchConfig: func(c *cliConfig) {
				c.readinessTimeout = -10 * time.Second
			},
			wantErr: errors.New("invalid readiness-timeout, should be >= 1"),
		},
		{
			desc:       "proxy enabled invalid prometheus local port",
			baseConfig: goodConfigWithProxy,
			patchConfig: func(c *cliConfig) {
				c.apiProxyPrometheusLocalPort = 0
			},
			wantErr: errors.New("invalid api-proxy-prometheus-local-port, should be > 0"),
		},
		{
			desc:       "proxy enabled invalid prometheus remote port",
			baseConfig: goodConfigWithProxy,
			patchConfig: func(c *cliConfig) {
				c.apiProxyPrometheusRemotePort = 0
			},
			wantErr: errors.New("invalid api-proxy-prometheus-remote-port, should be > 0"),
		},
		{
			desc:       "proxy enabled missing prometheus service name",
			baseConfig: goodConfigWithProxy,
			patchConfig: func(c *cliConfig) {
				c.apiProxyPrometheusServiceName = ""
			},
			wantErr: errors.New("missing api-proxy-prometheus-service-name"),
		},
	} {
		t.Run(testCase.desc, func(t *testing.T) {
			cfg := testCase.baseConfig

			testCase.patchConfig(&cfg)

			assert.Equal(
				t,
				testCase.wantErr,
				cfg.validateRuntimeConfig(),
			)

			if testCase.wantMemberID != "" {
				assert.Equal(
					t,
					testCase.wantMemberID,
					cfg.memberID,
				)
			}

		})
	}
}
