package config

import (
	"context"

	"github.com/imdario/mergo"
)

type LeaderChecker interface {
	IsLeader() bool
}

type Reconciler struct {
	sourcePath string
	outputPath string

	leaderChecker LeaderChecker
}

func NewReconciller(src, out string) *Reconciler {
	return &Reconciler{
		sourcePath: src,
		outputPath: out,
	}
}

func (r *Reconciler) SetLeaderChecker(lc LeaderChecker) {
	r.leaderChecker = lc
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	cfg, err := loadConfiguration(r.sourcePath)
	if err != nil {
		return err
	}

	targetCfg := cfg.Follower

	if cfg.Leader != nil && r.leaderChecker != nil && r.leaderChecker.IsLeader() {
		if err := mergo.Merge(
			targetCfg,
			cfg.Leader,
			mergo.WithOverride,
			mergo.WithAppendSlice,
		); err != nil {
			return err
		}
	}

	return writeConfiguration(r.outputPath, targetCfg)
}
