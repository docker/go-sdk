package wait_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/container/exec"
	"github.com/docker/go-sdk/container/wait"
)

type mockExecTarget struct {
	waitDuration time.Duration
	successAfter time.Time
	exitCode     int
	response     string
	failure      error
}

func (st mockExecTarget) Host(_ context.Context) (string, error) {
	return "", errors.New("not implemented")
}

func (st mockExecTarget) Inspect(_ context.Context) (client.ContainerInspectResult, error) {
	return client.ContainerInspectResult{}, errors.New("not implemented")
}

func (st mockExecTarget) MappedPort(_ context.Context, n network.Port) (network.Port, error) {
	return n, errors.New("not implemented")
}

func (st mockExecTarget) Logs(_ context.Context) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (st mockExecTarget) Exec(ctx context.Context, _ []string, _ ...exec.ProcessOption) (int, io.Reader, error) {
	var reader io.Reader
	if st.response != "" {
		reader = bytes.NewReader([]byte(st.response))
	}

	// Return success immediately once successAfter has passed, without sleeping.
	if !st.successAfter.IsZero() && time.Now().After(st.successAfter) {
		return 0, reader, nil
	}

	if st.waitDuration > 0 {
		select {
		case <-time.After(st.waitDuration):
		case <-ctx.Done():
			return st.exitCode, nil, ctx.Err()
		}
	}

	return st.exitCode, reader, st.failure
}

func (st mockExecTarget) State(_ context.Context) (*container.State, error) {
	return nil, errors.New("not implemented")
}

func (st mockExecTarget) CopyFromContainer(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (st mockExecTarget) Logger() *slog.Logger {
	return slog.Default()
}

func TestExecStrategyWaitUntilReady(t *testing.T) {
	target := mockExecTarget{}
	wg := wait.NewExecStrategy([]string{"true"}).
		WithTimeout(30 * time.Second)
	err := wg.WaitUntilReady(context.Background(), target)
	require.NoError(t, err)
}

func TestExecStrategyWaitUntilReadyForExec(t *testing.T) {
	target := mockExecTarget{}
	wg := wait.ForExec([]string{"true"})
	err := wg.WaitUntilReady(context.Background(), target)
	require.NoError(t, err)
}

func TestExecStrategyWaitUntilReady_MultipleChecks(t *testing.T) {
	target := mockExecTarget{
		exitCode:     10,
		successAfter: time.Now().Add(2 * time.Second),
	}
	wg := wait.NewExecStrategy([]string{"true"}).
		WithPollInterval(500 * time.Millisecond)
	err := wg.WaitUntilReady(context.Background(), target)
	require.NoError(t, err)
}

func TestExecStrategyWaitUntilReady_DeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	target := mockExecTarget{
		waitDuration: 1 * time.Second,
	}
	wg := wait.NewExecStrategy([]string{"true"})
	err := wg.WaitUntilReady(ctx, target)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestExecStrategyWaitUntilReady_CustomExitCode(t *testing.T) {
	target := mockExecTarget{
		exitCode: 10,
	}
	wg := wait.NewExecStrategy([]string{"true"}).WithExitCodeMatcher(func(exitCode int) bool {
		return exitCode == 10
	})
	err := wg.WaitUntilReady(context.Background(), target)
	require.NoError(t, err)
}

func TestExecStrategyWaitUntilReady_withExitCode(t *testing.T) {
	target := mockExecTarget{
		exitCode: 10,
	}
	wg := wait.NewExecStrategy([]string{"true"}).WithExitCode(10)
	// Default is 60. Let's shorten that
	wg.WithTimeout(time.Second * 2)
	err := wg.WaitUntilReady(context.Background(), target)
	require.NoError(t, err)

	// Ensure we aren't spuriously returning on any code
	wg = wait.NewExecStrategy([]string{"true"}).WithExitCode(0)
	wg.WithTimeout(time.Second * 2)
	err = wg.WaitUntilReady(context.Background(), target)
	require.Errorf(t, err, "Expected strategy to timeout out")
}

func TestExecStrategyWaitUntilReady_ExecPerPollTimeout(t *testing.T) {
	pollInterval := 100 * time.Millisecond
	target := mockExecTarget{
		exitCode:     1,
		waitDuration: 3 * pollInterval,            // exec takes longer than one poll tick
		successAfter: time.Now().Add(time.Second), // after 1s exec returns quickly
	}
	wg := wait.NewExecStrategy([]string{"true"}).
		WithPollInterval(pollInterval).
		WithTimeout(5 * time.Second)
	err := wg.WaitUntilReady(context.Background(), target)
	require.NoError(t, err)
}

func TestExecStrategyWaitUntilReady_ExecTimeoutDeadlineExceeded(t *testing.T) {
	pollInterval := 100 * time.Millisecond
	target := mockExecTarget{
		waitDuration: 10 * pollInterval, // always hangs longer than PollInterval
	}
	wg := wait.NewExecStrategy([]string{"true"}).
		WithPollInterval(pollInterval).
		WithTimeout(500 * time.Millisecond)
	err := wg.WaitUntilReady(context.Background(), target)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestExecStrategyWaitUntilReady_RetryOnError(t *testing.T) {
	target := mockExecTarget{
		failure:      errors.New("transient exec error"),
		successAfter: time.Now().Add(time.Second),
	}
	wg := wait.NewExecStrategy([]string{"true"}).
		WithPollInterval(100 * time.Millisecond).
		WithTimeout(5 * time.Second).
		WithRetryOnError()
	err := wg.WaitUntilReady(context.Background(), target)
	require.NoError(t, err)
}

func TestExecStrategyWaitUntilReady_FailOnError(t *testing.T) {
	execErr := errors.New("exec failed")
	target := mockExecTarget{
		failure: execErr,
	}
	wg := wait.NewExecStrategy([]string{"true"}).
		WithTimeout(5 * time.Second)
	err := wg.WaitUntilReady(context.Background(), target)
	require.ErrorIs(t, err, execErr)
}
