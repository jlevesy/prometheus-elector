package config

import (
	"bytes"
	"errors"
	"io"
	"os"

	"go.yaml.in/yaml/v3"
)

type config struct {
	Follower map[string]any `yaml:"follower"`
	Leader   map[string]any `yaml:"leader"`
}

func loadConfiguration(path string) (*config, error) {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg config

	decoder := yaml.NewDecoder(bytes.NewReader(fileBytes))
	decoder.KnownFields(true)

	if err = decoder.Decode(&cfg); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	if cfg.Follower == nil {
		return nil, errors.New("missing follower configuration")
	}

	return &cfg, nil
}

func writeConfiguration(path string, cfg map[string]any) error {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0600)
}
