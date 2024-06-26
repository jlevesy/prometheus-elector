package watcher

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"k8s.io/klog/v2"

	"crypto/sha256"
	"encoding/hex"

	"github.com/jlevesy/prometheus-elector/config"
	"github.com/jlevesy/prometheus-elector/notifier"
)

type FileWatcher struct {
	fsWatcher   *fsnotify.Watcher
	reconciler  *config.Reconciler
	notifier    notifier.Notifier
	configPaths []string
}

// Map to store the SHA-256 checksum of each file being monitored
var prevChecksumMap map[string]string

func New(configPaths []string, reconciler *config.Reconciler, notifier notifier.Notifier) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("unable to create fsnotify watcher: %w", err)
	}

	for _, path := range configPaths {
		dir := filepath.Dir(path)
		if err := watcher.Add(dir); err != nil {
			return nil, fmt.Errorf("unable to create watch config directory: %w", err)
		}
		klog.InfoS("Watching config directory", "path", dir)
	}

	return &FileWatcher{
		fsWatcher:   watcher,
		reconciler:  reconciler,
		notifier:    notifier,
		configPaths: configPaths,
	}, nil
}

func (f *FileWatcher) Watch(ctx context.Context) error {

	prevChecksumMap = make(map[string]string)
	// Record initial checksum
	for _, configPath := range f.configPaths {
		initialChecksum, err := getFileChecksum(configPath)
		if err != nil {
			klog.ErrorS(err, "Error getting initial checksum", "path", configPath)
			return err
		}
		klog.Info("Initial checksum for ", configPath, " is: ", initialChecksum)
		prevChecksumMap[configPath] = initialChecksum
	}
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				return nil
			}
			return ctx.Err()
		case evt, ok := <-f.fsWatcher.Events:
			if !ok {
				return nil
			}
			for _, configPath := range f.configPaths {
				configFileName := filepath.Base(configPath)
				if !isConfigFileChanged(evt, configFileName) {
					continue
				}
				currentChecksum, err := getFileChecksum(configPath)
				if err != nil {
					klog.ErrorS(err, "Error getting current checksum")
					return err
				}
				if !(currentChecksum != prevChecksumMap[configPath]) {
					continue
				}
				klog.Info("File currentChecksum for", configPath, " is: ", currentChecksum)
				klog.Info(configFileName, " file content has changed, reconciling")
				prevChecksumMap[configPath] = currentChecksum

				klog.Info("Configuration file ", evt.Name, " changed, reconciling...")

				if err := f.reconciler.Reconcile(ctx); err != nil {
					klog.ErrorS(err, "Reconciler reported an error")
					continue
				}

				if err := f.notifier.Notify(ctx); err != nil {
					klog.ErrorS(err, "Unable to notify prometheus")
					continue
				}
			}
		case err, ok := <-f.fsWatcher.Errors:
			if !ok {
				return nil
			}

			klog.ErrorS(err, "Watcher reported an error")
		}
	}
}

func isConfigFileChanged(evt fsnotify.Event, configFileName string) bool {

	if !(filepath.Base(evt.Name) == configFileName) {
		return false
	}
	if !(evt.Has(fsnotify.Create) || evt.Has(fsnotify.Chmod)) {
		return false
	}
	return true
}

// getFileChecksum returns the SHA-256 checksum of the file at the given path.
func getFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (f *FileWatcher) Close() error {
	return f.fsWatcher.Close()
}
