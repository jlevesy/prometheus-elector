package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"
)

type cliConfig struct {
	memberID           string
	leaseName          string
	leaseNamespace     string
	leaseDuration      time.Duration
	leaseRenewDeadline time.Duration
	leaseRetryPeriod   time.Duration

	kubeConfigPath string
	configPath     string

	outputPath string
	reloadURL  string
	init       bool
}

func newCLIConfig() cliConfig {
	return cliConfig{
		memberID: os.Getenv("POD_NAME"),
	}
}

func (c *cliConfig) validateInitConfig() error {
	if c.configPath == "" {
		return errors.New("missing config path")
	}

	if c.outputPath == "" {
		return errors.New("missing output path")
	}

	return nil
}

func (c *cliConfig) validateElectionConfig() error {
	if c.leaseName == "" {
		return errors.New("missing lease-name flag")
	}

	if c.leaseNamespace == "" {
		return errors.New("missing lease-namespace flag")
	}

	if c.reloadURL == "" {
		return errors.New("missing reloadURL path")
	}

	if c.memberID == "" {
		var err error

		c.memberID, err = os.Hostname()
		if err != nil {
			return fmt.Errorf("can't read hostname: %w", err)
		}
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
	flag.StringVar(&c.reloadURL, "reload-url", "", "URL to the reload configuration endpoint")
	flag.BoolVar(&c.init, "init", false, "Only init the prometheus config file")
}
