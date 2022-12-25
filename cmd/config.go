package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/http/httpguts"
)

type cliConfig struct {
	// Init config.

	// Are we running in init mode? (ie: writing the configuration once then exititing).
	init bool

	// Config and output paths.
	configPath string
	outputPath string

	// Runtime config.
	// Election setup.
	memberID           string
	leaseName          string
	leaseNamespace     string
	leaseDuration      time.Duration
	leaseRenewDeadline time.Duration
	leaseRetryPeriod   time.Duration

	// How to notify prometheus for an update.
	notifyHTTPURL          string
	notifyHTTPMethod       string
	notifyRetryMaxAttempts int
	notifyRetryDelay       time.Duration

	// API setup
	apiListenAddr         string
	apiShutdownGraceDelay time.Duration

	// Path to a kubeconfig (if running outside from the cluster).
	kubeConfigPath string
}

func newCLIConfig() cliConfig {
	return cliConfig{
		memberID: os.Getenv("POD_NAME"),
	}
}

func (c *cliConfig) validateInitConfig() error {
	if c.configPath == "" {
		return errors.New("missing config flag")
	}

	if c.outputPath == "" {
		return errors.New("missing output flag")
	}

	return nil
}

func (c *cliConfig) validateRuntimeConfig() error {
	if c.leaseName == "" {
		return errors.New("missing lease-name flag")
	}

	if c.leaseNamespace == "" {
		return errors.New("missing lease-namespace flag")
	}

	if c.memberID == "" {
		var err error

		c.memberID, err = os.Hostname()
		if err != nil {
			return fmt.Errorf("can't read hostname: %w", err)
		}
	}

	if c.notifyHTTPURL == "" {
		return errors.New("missing notify-http-url flag")
	}

	if !validHTTPMethod(c.notifyHTTPMethod) {
		return errors.New("invalid notify-http-method")
	}

	if c.notifyRetryMaxAttempts < 1 {
		return errors.New("invalid notify-retry-max-attempts, should be >= 1")
	}

	if c.notifyRetryDelay < 1 {
		return errors.New("invalid notify-retry-delay, should be >= 1")
	}

	if c.apiListenAddr == "" {
		return errors.New("missing api-listen-address")
	}

	if c.apiShutdownGraceDelay < 0 {
		return errors.New("invalid api-shudown-grace-delay, should be >= 0")
	}

	return nil
}

func (c *cliConfig) setupFlags() {
	flag.StringVar(&c.leaseName, "lease-name", "", "Name of lease lock")
	flag.StringVar(&c.leaseNamespace, "lease-namespace", "", "Name of lease lock namespace")
	flag.DurationVar(&c.leaseDuration, "lease-duration", 15*time.Second, "Duration of a lease, client wait the full duration of a lease before trying to take it over")
	flag.DurationVar(&c.leaseRenewDeadline, "lease-renew-deadline", 10*time.Second, "Maximum duration spent trying to renew the lease")
	flag.DurationVar(&c.leaseRetryPeriod, "lease-retry-period", 2*time.Second, "Delay between two attempts of taking/renewing the lease")
	flag.StringVar(&c.kubeConfigPath, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&c.configPath, "config", "", "Path of the prometheus-elector configuration")
	flag.StringVar(&c.outputPath, "output", "", "Path to write the active prometheus configuration")
	flag.StringVar(&c.notifyHTTPURL, "notify-http-url", "", "URL to the reload configuration endpoint")
	flag.StringVar(&c.notifyHTTPMethod, "notify-http-method", http.MethodPost, "HTTP method to use when sending the reload config request.")
	flag.IntVar(&c.notifyRetryMaxAttempts, "notify-retry-max-attempts", 5, "How many times to retry notifying prometheus on failure.")
	flag.DurationVar(&c.notifyRetryDelay, "notify-retry-delay", 10*time.Second, "How much time to wait between two notify retries.")
	flag.BoolVar(&c.init, "init", false, "Only init the prometheus config file")
	flag.StringVar(&c.apiListenAddr, "api-listen-address", ":9095", "HTTP listen address to use for the API.")
	flag.DurationVar(&c.apiShutdownGraceDelay, "api-shutdown-grace-delay", 15*time.Second, "Grace delay to apply when shutting down the API server")
}

// this is how the http standard library validates the method in NewRequestWithContext.
func validHTTPMethod(method string) bool {
	return len(method) > 0 && strings.IndexFunc(method, func(r rune) bool {
		return !httpguts.IsTokenRune(r)
	}) == -1
}
