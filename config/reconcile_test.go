package config_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jlevesy/prometheus-elector/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fileName = "prometheus.yaml"

func TestReconciler(t *testing.T) {
	for _, testCase := range []struct {
		desc           string
		isLeader       bool
		inputPath      string
		wantError      error
		wantResultPath string
	}{
		{
			desc:           "follower",
			inputPath:      "./testdata/config.yaml",
			isLeader:       false,
			wantResultPath: "./testdata/follower_no_leader_result.yaml",
		},
		{
			desc:           "leader",
			inputPath:      "./testdata/config.yaml",
			isLeader:       true,
			wantResultPath: "./testdata/leader_result.yaml",
		},
		{
			desc:           "no leader section",
			inputPath:      "./testdata/config_no_leader.yaml",
			wantResultPath: "./testdata/config_no_leader_result.yaml",
		},
		{
			desc:      "no follower section",
			inputPath: "./testdata/config_no_follower.yaml",
			wantError: errors.New("missing follower configuration"),
		},
	} {
		t.Run(testCase.desc, func(t *testing.T) {
			var (
				ctx        = context.Background()
				destDir    = t.TempDir()
				outPath    = filepath.Join(destDir, fileName)
				reconciler = config.NewReconciller(
					testCase.inputPath,
					outPath,
				)
			)

			err := reconciler.Reconcile(ctx, testCase.isLeader)
			if testCase.wantError != nil {
				assert.Equal(t, testCase.wantError, err)
				return
			}
			require.NoError(t, err)

			gotBytes, err := os.ReadFile(outPath)
			require.NoError(t, err)

			wantBytes, err := os.ReadFile(testCase.wantResultPath)
			require.NoError(t, err)

			assert.Equal(t, string(wantBytes), string(gotBytes))
		})
	}
}
