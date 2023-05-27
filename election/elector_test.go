package election_test

import (
	"context"
	"testing"
	"time"

	"github.com/jlevesy/prometheus-elector/election"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/leaderelection"
)

func TestElector(t *testing.T) {
	var (
		ctx        = context.Background()
		kubeClient = kubefake.NewSimpleClientset()
		config     = election.Config{
			LeaseName:      "test",
			LeaseNamespace: "test",
			MemberID:       "foo",
			LeaseDuration:  time.Second,
			RenewDeadline:  500 * time.Millisecond,
			RetryPeriod:    200 * time.Millisecond,
		}
		startedLeading = make(chan struct{}, 1)
		stoppedLeading = make(chan struct{}, 1)
	)

	elector, err := election.New(
		config,
		kubeClient,
		leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				startedLeading <- struct{}{}
			},
			OnStoppedLeading: func() {
				stoppedLeading <- struct{}{}
			},
		},
		nil, // nil metrics registry. We don't really care about them in this test.
	)
	require.NoError(t, err)

	defer func() {
		_ = elector.Stop(ctx)
	}()

	for i := 0; i < 100; i++ {
		err = elector.Start(ctx)
		require.NoError(t, err)

		err = elector.Start(ctx)
		assert.Equal(t, election.ErrAlreadyRunning, err)

		<-startedLeading
		assert.True(t, elector.Status().IsLeader())
		assert.Equal(t, "foo", elector.Status().GetLeader())

		err = elector.Stop(ctx)
		require.NoError(t, err)

		err = elector.Stop(ctx)
		assert.Equal(t, election.ErrNotRunning, err)

		<-stoppedLeading
		assert.False(t, elector.Status().IsLeader())
		assert.Equal(t, "", elector.Status().GetLeader())
	}
}
