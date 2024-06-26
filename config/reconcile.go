package config

import (
	"context"

	"github.com/imdario/mergo"
	"github.com/jlevesy/prometheus-elector/election"
)

type Reconciler struct {
	sourcePath    string
	outputPath    string
	leaderPath    string
	leaderChecker election.LeaderChecker
}

func NewReconciller(src, out, leader string) *Reconciler {
	return &Reconciler{
		sourcePath: src,
		outputPath: out,
		leaderPath: leader,
	}
}

func (r *Reconciler) SetLeaderChecker(lc election.LeaderChecker) {
	r.leaderChecker = lc
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	cfg, err := loadConfiguration(r.sourcePath, r.leaderPath)
	if err != nil {
		return err
	}

	targetCfg := cfg.Follower

	if cfg.Leader != nil && r.leaderChecker != nil && r.leaderChecker.IsLeader() {
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
