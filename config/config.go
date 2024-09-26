package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type config struct {
	Follower map[string]any
	Leader   map[string]any
}

func loadConfiguration(followerPath string, leaderPath string) (*config, error) {
	followerFileBytes, err := os.ReadFile(followerPath)
	if err != nil {
		return nil, err
	}
	leaderFileBytes, err := os.ReadFile(leaderPath)
	if err != nil {
		return nil, err
	}

	var cfg config

	if err = yaml.UnmarshalStrict(followerFileBytes, &cfg.Follower); err != nil {
		return nil, err
	}

	if cfg.Follower == nil {
		return nil, errors.New("Missing follower configuration")
	}

	if err := yaml.UnmarshalStrict(leaderFileBytes, &cfg.Leader); err != nil {
		fmt.Println("Error unmarshalling Leader configuration:", err)
		return nil, err
	}
	if cfg.Leader == nil {
		fmt.Println("Error: ", errors.New("Missing leader configuration"))
		return nil, err
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
