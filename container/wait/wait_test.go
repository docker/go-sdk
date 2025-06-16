package wait

import (
	"context"
	"errors"
	"io"
	slog "log/slog"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-sdk/container/exec"
)

var ErrPortNotFound = errors.New("port not found")

type MockStrategyTarget struct {
	HostImpl              func(context.Context) (string, error)
	InspectImpl           func(context.Context) (*container.InspectResponse, error)
	PortsImpl             func(context.Context) (nat.PortMap, error)
	MappedPortImpl        func(context.Context, nat.Port) (nat.Port, error)
	LogsImpl              func(context.Context) (io.ReadCloser, error)
	ExecImpl              func(context.Context, []string, ...exec.ProcessOption) (int, io.Reader, error)
	StateImpl             func(context.Context) (*container.State, error)
	CopyFromContainerImpl func(context.Context, string) (io.ReadCloser, error)
	LoggerImpl            func() *slog.Logger
}

func (st *MockStrategyTarget) Host(ctx context.Context) (string, error) {
	return st.HostImpl(ctx)
}

func (st *MockStrategyTarget) Inspect(ctx context.Context) (*container.InspectResponse, error) {
	return st.InspectImpl(ctx)
}

func (st *MockStrategyTarget) MappedPort(ctx context.Context, port nat.Port) (nat.Port, error) {
	return st.MappedPortImpl(ctx, port)
}

func (st *MockStrategyTarget) Logs(ctx context.Context) (io.ReadCloser, error) {
	return st.LogsImpl(ctx)
}

func (st *MockStrategyTarget) Exec(ctx context.Context, cmd []string, options ...exec.ProcessOption) (int, io.Reader, error) {
	return st.ExecImpl(ctx, cmd, options...)
}

func (st *MockStrategyTarget) State(ctx context.Context) (*container.State, error) {
	return st.StateImpl(ctx)
}

func (st *MockStrategyTarget) CopyFromContainer(ctx context.Context, filePath string) (io.ReadCloser, error) {
	return st.CopyFromContainerImpl(ctx, filePath)
}

func (st *MockStrategyTarget) Logger() *slog.Logger {
	return st.LoggerImpl()
}
