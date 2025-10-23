package wait

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"github.com/docker/go-sdk/container/exec"
)

var ErrPortNotFound = errors.New("port not found")

type MockStrategyTarget struct {
	HostImpl              func(context.Context) (string, error)
	InspectImpl           func(context.Context) (client.ContainerInspectResult, error)
	PortsImpl             func(context.Context) (network.PortMap, error)
	MappedPortImpl        func(context.Context, network.Port) (network.Port, error)
	LogsImpl              func(context.Context) (io.ReadCloser, error)
	ExecImpl              func(context.Context, []string, ...exec.ProcessOption) (int, io.Reader, error)
	StateImpl             func(context.Context) (*container.State, error)
	CopyFromContainerImpl func(context.Context, string) (io.ReadCloser, error)
	LoggerImpl            func() *slog.Logger
}

func (st *MockStrategyTarget) Host(ctx context.Context) (string, error) {
	return st.HostImpl(ctx)
}

func (st *MockStrategyTarget) Inspect(ctx context.Context) (client.ContainerInspectResult, error) {
	return st.InspectImpl(ctx)
}

func (st *MockStrategyTarget) MappedPort(ctx context.Context, port network.Port) (network.Port, error) {
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
