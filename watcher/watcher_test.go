package watcher_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jlevesy/prometheus-elector/config"
	"github.com/jlevesy/prometheus-elector/watcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const defaultConfig = `
leader:
  remote_write:
  - url: http://12.3.4.5

follower:
  scrape_configs:
  - job_name: 'foobar'
    scrape_interval: 5s
    static_configs:
    - targets: ['localhost:8080']
`

const wantConfig = `global:
  scrape_interval: 1m
  scrape_timeout: 10s
  evaluation_interval: 1m
scrape_configs:
- job_name: foobar
  honor_timestamps: true
  scrape_interval: 5s
  scrape_timeout: 5s
  metrics_path: /metrics
  scheme: http
  follow_redirects: true
  enable_http2: true
  static_configs:
  - targets:
    - localhost:8080
`

const (
	fileName     = "config.yaml"
	destFileName = "result.yaml"
)

func TestFileWatcher(t *testing.T) {
	var (
		dir        = t.TempDir()
		configPath = filepath.Join(dir, fileName)
		destPath   = filepath.Join(dir, destFileName)

		reconciler = config.NewReconciller(configPath, destPath)

		notifiedCh = make(chan struct{})
		notifier   = func() error {
			notifiedCh <- struct{}{}
			return nil
		}

		ctx, cancel = context.WithCancel(context.Background())
	)

	defer cancel()

	err := simulateConfigmapWrite(dir, fileName, []byte(defaultConfig))
	require.NoError(t, err)

	watcher, err := watcher.New(dir, reconciler, notifierFunc(notifier))
	require.NoError(t, err)

	defer watcher.Close()

	go func() {
		err := watcher.Watch(ctx)
		require.NoError(t, err)
	}()

	go func() {
		err := simulateConfigmapWrite(dir, fileName, []byte(defaultConfig))
		require.NoError(t, err)
	}()

	<-notifiedCh

	gotConfig, err := os.ReadFile(destPath)
	require.NoError(t, err)

	assert.Equal(t, wantConfig, string(gotConfig))
}

// Vague attempt to simulate a full configmap write in k8s.
// See https://github.com/kubernetes/kubernetes/blob/master/pkg/volume/util/atomic_writer.go#L128 for the full implementation.
func simulateConfigmapWrite(basePath, fileName string, payload []byte) error {
	var (
		dataDir  = filepath.Join(basePath, "..data")
		filePath = filepath.Join(basePath, fileName)
	)

	if _, err := os.Stat(dataDir); err == nil {
		if err := os.RemoveAll(dataDir); err != nil {
			return err
		}
	}

	if err := os.WriteFile(filePath, payload, 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	return nil
}

type notifierFunc func() error

func (n notifierFunc) Notify(context.Context) error {
	return n()
}
