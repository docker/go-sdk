package wait

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-sdk/dockercontainer/exec"
)

var (
	_          Strategy        = (*NopStrategy)(nil)
	_          StrategyTimeout = (*NopStrategy)(nil)
	noopLogger                 = slog.New(slog.NewTextHandler(io.Discard, nil))
)

type NopStrategy struct {
	timeout        *time.Duration
	waitUntilReady func(context.Context, StrategyTarget) error
}

func ForNop(
	waitUntilReady func(context.Context, StrategyTarget) error,
) *NopStrategy {
	return &NopStrategy{
		waitUntilReady: waitUntilReady,
	}
}

func (ws *NopStrategy) Timeout() *time.Duration {
	return ws.timeout
}

func (ws *NopStrategy) WithTimeout(timeout time.Duration) *NopStrategy {
	ws.timeout = &timeout
	return ws
}

func (ws *NopStrategy) WaitUntilReady(ctx context.Context, target StrategyTarget) error {
	return ws.waitUntilReady(ctx, target)
}

type NopStrategyTarget struct {
	ReaderCloser   io.ReadCloser
	ContainerState container.State
}

func (st *NopStrategyTarget) Host(_ context.Context) (string, error) {
	return "", nil
}

func (st *NopStrategyTarget) Inspect(_ context.Context) (*container.InspectResponse, error) {
	return nil, nil
}

func (st *NopStrategyTarget) MappedPort(_ context.Context, n nat.Port) (nat.Port, error) {
	return n, nil
}

func (st *NopStrategyTarget) Logs(_ context.Context) (io.ReadCloser, error) {
	return st.ReaderCloser, nil
}

func (st *NopStrategyTarget) Exec(_ context.Context, _ []string, _ ...exec.ProcessOption) (int, io.Reader, error) {
	return 0, nil, nil
}

func (st *NopStrategyTarget) State(_ context.Context) (*container.State, error) {
	return &st.ContainerState, nil
}

func (st *NopStrategyTarget) CopyFromContainer(_ context.Context, _ string) (io.ReadCloser, error) {
	return st.ReaderCloser, nil
}

func (st *NopStrategyTarget) Logger() *slog.Logger {
	return noopLogger
}
