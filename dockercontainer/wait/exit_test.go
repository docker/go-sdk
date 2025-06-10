package wait

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-sdk/dockercontainer/exec"
)

type exitStrategyTarget struct {
	isRunning bool
}

func (st *exitStrategyTarget) Host(_ context.Context) (string, error) {
	return "", nil
}

func (st *exitStrategyTarget) Inspect(_ context.Context) (*container.InspectResponse, error) {
	return nil, nil
}

func (st *exitStrategyTarget) MappedPort(_ context.Context, n nat.Port) (nat.Port, error) {
	return n, nil
}

func (st *exitStrategyTarget) Logs(_ context.Context) (io.ReadCloser, error) {
	return nil, nil
}

func (st *exitStrategyTarget) Exec(_ context.Context, _ []string, _ ...exec.ProcessOption) (int, io.Reader, error) {
	return 0, nil, nil
}

func (st *exitStrategyTarget) State(_ context.Context) (*container.State, error) {
	return &container.State{Running: st.isRunning}, nil
}

func (st *exitStrategyTarget) CopyFromContainer(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

func (st *exitStrategyTarget) Logger() *slog.Logger {
	return slog.Default()
}

func TestWaitForExit(t *testing.T) {
	target := exitStrategyTarget{
		isRunning: false,
	}
	wg := NewExitStrategy().WithExitTimeout(100 * time.Millisecond)
	err := wg.WaitUntilReady(context.Background(), &target)
	require.NoError(t, err)
}
