package config

import (
	"context"

	"github.com/imdario/mergo"
)

type Reconciler struct {
	sourcePath string
	outputPath string
}

func NewReconciller(src, out string) *Reconciler {
	return &Reconciler{
		sourcePath: src,
		outputPath: out,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, leader bool) error {
	cfg, err := loadConfiguration(r.sourcePath)
	if err != nil {
		return err
	}

	targetCfg := cfg.Follower

	if leader {
		if err := mergo.Merge(
			&targetCfg,
			cfg.Leader,
			mergo.WithOverride,
			mergo.WithAppendSlice,
		); err != nil {
			return err
		}
	}

	return writeConfiguration(r.outputPath, targetCfg)
}
