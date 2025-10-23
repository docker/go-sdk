package wait

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/container/exec"
)

func TestWaitForListeningPortSucceeds(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()

	rawPort := listener.Addr().(*net.TCPAddr).Port
	port, ok := network.PortFrom(uint16(rawPort), "tcp")
	require.True(t, ok)

	var mappedPortCount, execCount int
	target := &MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			defer func() { mappedPortCount++ }()
			if mappedPortCount == 0 {
				return network.Port{}, ErrPortNotFound
			}
			return port, nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Running: true,
			}, nil
		},
		ExecImpl: func(_ context.Context, _ []string, _ ...exec.ProcessOption) (int, io.Reader, error) {
			defer func() { execCount++ }()
			if execCount == 0 {
				return 1, nil, nil
			}
			return 0, nil, nil
		},
		LoggerImpl: slog.Default,
	}

	wg := ForListeningPort(network.MustParsePort("80")).
		WithTimeout(5 * time.Second).
		WithPollInterval(100 * time.Millisecond)

	err = wg.WaitUntilReady(context.Background(), target)
	require.NoError(t, err)
}

func TestWaitForListeningPortInternallySucceeds(t *testing.T) {
	localPort := network.MustParsePort("80/tcp")
	mappedPort := network.MustParsePort("8080/tcp")

	var mappedPortCount, execCount int
	target := &MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, p network.Port) (network.Port, error) {
			if p.Num() != localPort.Num() {
				return network.Port{}, ErrPortNotFound
			}
			defer func() { mappedPortCount++ }()
			if mappedPortCount <= 2 {
				return network.Port{}, ErrPortNotFound
			}
			return mappedPort, nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Running: true,
			}, nil
		},
		ExecImpl: func(_ context.Context, _ []string, _ ...exec.ProcessOption) (int, io.Reader, error) {
			defer func() { execCount++ }()
			if execCount <= 2 {
				return 1, nil, nil
			}
			return 0, nil, nil
		},
		LoggerImpl: slog.Default,
	}

	wg := ForListeningPort(localPort).
		SkipExternalCheck().
		WithTimeout(5 * time.Second).
		WithPollInterval(100 * time.Millisecond)

	err := wg.WaitUntilReady(context.Background(), target)
	require.NoError(t, err)
}

func TestWaitForMappedPortSucceeds(t *testing.T) {
	localPort := network.MustParsePort("80/tcp")
	mappedPort := network.MustParsePort("8080/tcp")

	var mappedPortCount int
	target := &MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, p network.Port) (network.Port, error) {
			if p.Num() != localPort.Num() {
				return network.Port{}, ErrPortNotFound
			}
			defer func() { mappedPortCount++ }()
			if mappedPortCount <= 2 {
				return network.Port{}, ErrPortNotFound
			}
			return mappedPort, nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Running: true,
			}, nil
		},
		LoggerImpl: slog.Default,
	}

	wg := ForMappedPort(localPort).
		WithTimeout(5 * time.Second).
		WithPollInterval(100 * time.Millisecond)

	err := wg.WaitUntilReady(context.Background(), target)
	require.NoError(t, err)
}

func TestWaitForExposedPortSkipChecksSucceeds(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()

	rawPort := listener.Addr().(*net.TCPAddr).Port
	port, ok := network.PortFrom(uint16(rawPort), "tcp")
	require.True(t, ok)

	var inspectCount, mappedPortCount, execCount int
	target := &MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		InspectImpl: func(_ context.Context) (client.ContainerInspectResult, error) {
			defer func() { inspectCount++ }()
			if inspectCount == 0 {
				// Simulate a container that hasn't bound any ports yet.
				return client.ContainerInspectResult{
					Container: container.InspectResponse{
						NetworkSettings: &container.NetworkSettings{
							Ports: network.PortMap{},
						},
					},
				}, nil
			}

			return client.ContainerInspectResult{
				Container: container.InspectResponse{
					NetworkSettings: &container.NetworkSettings{
						Ports: network.PortMap{
							network.MustParsePort("80"): []network.PortBinding{
								{
									HostIP:   netip.IPv4Unspecified(),
									HostPort: port.String(),
								},
							},
						},
					},
				},
			}, nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			defer func() { mappedPortCount++ }()
			if mappedPortCount == 0 {
				return network.Port{}, ErrPortNotFound
			}
			return port, nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Running: true,
			}, nil
		},
		ExecImpl: func(_ context.Context, _ []string, _ ...exec.ProcessOption) (int, io.Reader, error) {
			defer func() { execCount++ }()
			if execCount == 0 {
				return 1, nil, nil
			}
			return 0, nil, nil
		},
		LoggerImpl: slog.Default,
	}

	wg := ForExposedPort().
		WithTimeout(5 * time.Second).
		WithPollInterval(100 * time.Millisecond)

	err = wg.WaitUntilReady(context.Background(), target)
	require.NoError(t, err)
}

func TestHostPortStrategyFailsWhileGettingPortDueToOOMKilledContainer(t *testing.T) {
	var mappedPortCount int
	target := &MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			defer func() { mappedPortCount++ }()
			if mappedPortCount == 0 {
				return network.Port{}, ErrPortNotFound
			}
			return network.MustParsePort("49152"), nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				OOMKilled: true,
			}, nil
		},
	}

	wg := NewHostPortStrategy(network.MustParsePort("80")).
		WithTimeout(5 * time.Second).
		WithPollInterval(100 * time.Millisecond)

	{
		err := wg.WaitUntilReady(context.Background(), target)
		require.ErrorContains(t, err, "container crashed with out-of-memory (OOMKilled)")
	}
}

func TestHostPortStrategyFailsWhileGettingPortDueToExitedContainer(t *testing.T) {
	var mappedPortCount int
	target := &MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			defer func() { mappedPortCount++ }()
			if mappedPortCount == 0 {
				return network.Port{}, ErrPortNotFound
			}
			return network.MustParsePort("49152"), nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Status:   "exited",
				ExitCode: 1,
			}, nil
		},
	}

	wg := NewHostPortStrategy(network.MustParsePort("80")).
		WithTimeout(5 * time.Second).
		WithPollInterval(100 * time.Millisecond)

	{
		err := wg.WaitUntilReady(context.Background(), target)
		require.ErrorContains(t, err, "container exited with code 1")
	}
}

func TestHostPortStrategyFailsWhileGettingPortDueToUnexpectedContainerStatus(t *testing.T) {
	var mappedPortCount int
	target := &MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			defer func() { mappedPortCount++ }()
			if mappedPortCount == 0 {
				return network.Port{}, ErrPortNotFound
			}
			return network.MustParsePort("49152"), nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Status: "dead",
			}, nil
		},
	}

	wg := NewHostPortStrategy(network.MustParsePort("80")).
		WithTimeout(5 * time.Second).
		WithPollInterval(100 * time.Millisecond)

	{
		err := wg.WaitUntilReady(context.Background(), target)
		require.ErrorContains(t, err, "unexpected container status \"dead\"")
	}
}

func TestHostPortStrategyFailsWhileExternalCheckingDueToOOMKilledContainer(t *testing.T) {
	target := &MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			return network.MustParsePort("49152"), nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				OOMKilled: true,
			}, nil
		},
	}

	wg := NewHostPortStrategy(network.MustParsePort("80")).
		WithTimeout(5 * time.Second).
		WithPollInterval(100 * time.Millisecond)

	{
		err := wg.WaitUntilReady(context.Background(), target)
		require.ErrorContains(t, err, "container crashed with out-of-memory (OOMKilled)")
	}
}

func TestHostPortStrategyFailsWhileExternalCheckingDueToExitedContainer(t *testing.T) {
	target := &MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			return network.MustParsePort("49152"), nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Status:   "exited",
				ExitCode: 1,
			}, nil
		},
	}

	wg := NewHostPortStrategy(network.MustParsePort("80")).
		WithTimeout(5 * time.Second).
		WithPollInterval(100 * time.Millisecond)

	{
		err := wg.WaitUntilReady(context.Background(), target)
		require.ErrorContains(t, err, "container exited with code 1")
	}
}

func TestHostPortStrategyFailsWhileExternalCheckingDueToUnexpectedContainerStatus(t *testing.T) {
	target := &MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			return network.MustParsePort("49152"), nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Status: "dead",
			}, nil
		},
	}

	wg := NewHostPortStrategy(network.MustParsePort("80")).
		WithTimeout(5 * time.Second).
		WithPollInterval(100 * time.Millisecond)

	{
		err := wg.WaitUntilReady(context.Background(), target)
		require.ErrorContains(t, err, "unexpected container status \"dead\"")
	}
}

func TestHostPortStrategyFailsWhileInternalCheckingDueToOOMKilledContainer(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()

	rawPort := listener.Addr().(*net.TCPAddr).Port
	port, ok := network.PortFrom(uint16(rawPort), "tcp")
	require.True(t, ok)

	var stateCount int
	target := &MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			return port, nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			defer func() { stateCount++ }()
			if stateCount == 0 {
				return &container.State{
					Running: true,
				}, nil
			}
			return &container.State{
				OOMKilled: true,
			}, nil
		},
	}

	wg := NewHostPortStrategy(network.MustParsePort("80")).
		WithTimeout(5 * time.Second).
		WithPollInterval(100 * time.Millisecond)

	{
		err := wg.WaitUntilReady(context.Background(), target)
		require.ErrorContains(t, err, "container crashed with out-of-memory (OOMKilled)")
	}
}

func TestHostPortStrategyFailsWhileInternalCheckingDueToExitedContainer(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()

	rawPort := listener.Addr().(*net.TCPAddr).Port
	port, ok := network.PortFrom(uint16(rawPort), "tcp")
	require.True(t, ok)

	var stateCount int
	target := &MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			return port, nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			defer func() { stateCount++ }()
			if stateCount == 0 {
				return &container.State{
					Running: true,
				}, nil
			}
			return &container.State{
				Status:   "exited",
				ExitCode: 1,
			}, nil
		},
	}

	wg := NewHostPortStrategy(network.MustParsePort("80")).
		WithTimeout(5 * time.Second).
		WithPollInterval(100 * time.Millisecond)

	{
		err := wg.WaitUntilReady(context.Background(), target)
		require.ErrorContains(t, err, "container exited with code 1")
	}
}

func TestHostPortStrategyFailsWhileInternalCheckingDueToUnexpectedContainerStatus(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()

	rawPort := listener.Addr().(*net.TCPAddr).Port
	port, ok := network.PortFrom(uint16(rawPort), "tcp")
	require.True(t, ok)

	var stateCount int
	target := &MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			return port, nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			defer func() { stateCount++ }()
			if stateCount == 0 {
				return &container.State{
					Running: true,
				}, nil
			}
			return &container.State{
				Status: "dead",
			}, nil
		},
	}

	wg := NewHostPortStrategy(network.MustParsePort("80")).
		WithTimeout(5 * time.Second).
		WithPollInterval(100 * time.Millisecond)

	{
		err := wg.WaitUntilReady(context.Background(), target)
		require.ErrorContains(t, err, "unexpected container status \"dead\"")
	}
}

func TestHostPortStrategySucceedsGivenShellIsNotInstalled(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()

	rawPort := listener.Addr().(*net.TCPAddr).Port
	port, ok := network.PortFrom(uint16(rawPort), "tcp")
	require.True(t, ok)

	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, nil))

	target := &MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		InspectImpl: func(_ context.Context) (client.ContainerInspectResult, error) {
			return client.ContainerInspectResult{
				Container: container.InspectResponse{
					NetworkSettings: &container.NetworkSettings{
						Ports: network.PortMap{
							network.MustParsePort("80"): []network.PortBinding{
								{
									HostIP:   netip.IPv4Unspecified(),
									HostPort: port.String(),
								},
							},
						},
					},
				},
			}, nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			return port, nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Running: true,
			}, nil
		},
		ExecImpl: func(_ context.Context, _ []string, _ ...exec.ProcessOption) (int, io.Reader, error) {
			// This is the error that would be returned if the shell is not installed.
			return exitEaccess, nil, nil
		},
		LoggerImpl: func() *slog.Logger {
			return logger
		},
	}

	wg := NewHostPortStrategy(network.MustParsePort("80")).
		WithTimeout(5 * time.Second).
		WithPollInterval(100 * time.Millisecond)

	err = wg.WaitUntilReady(context.Background(), target)
	require.NoError(t, err)

	require.Contains(t, buf.String(), "Shell not executable in container, only external port validated")
}

func TestHostPortStrategySucceedsGivenShellIsNotFound(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()

	rawPort := listener.Addr().(*net.TCPAddr).Port
	port, ok := network.PortFrom(uint16(rawPort), "tcp")
	require.True(t, ok)

	bufLogger := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(bufLogger, nil))

	target := &MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		InspectImpl: func(_ context.Context) (client.ContainerInspectResult, error) {
			return client.ContainerInspectResult{
				Container: container.InspectResponse{
					NetworkSettings: &container.NetworkSettings{
						Ports: network.PortMap{
							network.MustParsePort("80"): []network.PortBinding{
								{
									HostIP:   netip.IPv4Unspecified(),
									HostPort: port.String(),
								},
							},
						},
					},
				},
			}, nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			return port, nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Running: true,
			}, nil
		},
		ExecImpl: func(_ context.Context, _ []string, _ ...exec.ProcessOption) (int, io.Reader, error) {
			// This is the error that would be returned if the shell is not found.
			return exitCmdNotFound, nil, nil
		},
		LoggerImpl: func() *slog.Logger {
			return logger
		},
	}

	wg := NewHostPortStrategy(network.MustParsePort("80")).
		WithTimeout(5 * time.Second).
		WithPollInterval(100 * time.Millisecond)

	err = wg.WaitUntilReady(context.Background(), target)
	require.NoError(t, err)

	require.Contains(t, bufLogger.String(), "Shell not found in container")
}
