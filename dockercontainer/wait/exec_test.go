package wait_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-sdk/dockercontainer/exec"
	"github.com/docker/go-sdk/dockercontainer/wait"
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

func (st mockExecTarget) Inspect(_ context.Context) (*container.InspectResponse, error) {
	return nil, errors.New("not implemented")
}

func (st mockExecTarget) MappedPort(_ context.Context, n nat.Port) (nat.Port, error) {
	return n, errors.New("not implemented")
}

func (st mockExecTarget) Logs(_ context.Context) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (st mockExecTarget) Exec(ctx context.Context, _ []string, _ ...exec.ProcessOption) (int, io.Reader, error) {
	time.Sleep(st.waitDuration)

	var reader io.Reader
	if st.response != "" {
		reader = bytes.NewReader([]byte(st.response))
	}

	if err := ctx.Err(); err != nil {
		return st.exitCode, reader, err
	}

	if !st.successAfter.IsZero() && time.Now().After(st.successAfter) {
		return 0, reader, st.failure
	}

	return st.exitCode, reader, st.failure
}

func (st mockExecTarget) State(_ context.Context) (*container.State, error) {
	return nil, errors.New("not implemented")
}

func (st mockExecTarget) CopyFromContainer(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
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
