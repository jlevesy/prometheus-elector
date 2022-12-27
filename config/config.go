package config

import (
	"errors"
	"os"

	promconfig "github.com/prometheus/prometheus/config"
	_ "github.com/prometheus/prometheus/plugins"
	"gopkg.in/yaml.v2"
)

type config struct {
	Follower *promconfig.Config `yaml:"follower"`
	Leader   *promconfig.Config `yaml:"leader"`
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

func writeConfiguration(path string, cfg *promconfig.Config) error {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0600)
}
