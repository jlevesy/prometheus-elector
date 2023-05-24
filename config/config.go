package config

import (
	"errors"
	"os"

	"gopkg.in/yaml.v2"
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

	if err = yaml.UnmarshalStrict(fileBytes, &cfg); err != nil {
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
