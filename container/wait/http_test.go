package wait_test

import (
	"context"
	_ "embed"
	"net/netip"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/container/wait"
)

func TestHttpStrategyFailsWhileGettingPortDueToOOMKilledContainer(t *testing.T) {
	var mappedPortCount int
	target := &wait.MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			defer func() { mappedPortCount++ }()
			if mappedPortCount == 0 {
				return network.Port{}, wait.ErrPortNotFound
			}
			return network.MustParsePort("49152"), nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				OOMKilled: true,
			}, nil
		},
		InspectImpl: func(_ context.Context) (client.ContainerInspectResult, error) {
			return client.ContainerInspectResult{
				Container: container.InspectResponse{
					NetworkSettings: &container.NetworkSettings{
						Ports: network.PortMap{
							network.MustParsePort("8080/tcp"): []network.PortBinding{
								{
									HostIP:   netip.MustParseAddr("127.0.0.1"),
									HostPort: "49152",
								},
							},
						},
					},
				},
			}, nil
		},
	}

	wg := wait.ForHTTP("/").
		WithTimeout(500 * time.Millisecond).
		WithPollInterval(100 * time.Millisecond)

	err := wg.WaitUntilReady(context.Background(), target)
	expected := "container crashed with out-of-memory (OOMKilled)"
	require.EqualError(t, err, expected)
}

func TestHttpStrategyFailsWhileGettingPortDueToExitedContainer(t *testing.T) {
	var mappedPortCount int
	target := &wait.MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			defer func() { mappedPortCount++ }()
			if mappedPortCount == 0 {
				return network.Port{}, wait.ErrPortNotFound
			}
			return network.MustParsePort("49152"), nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Status:   "exited",
				ExitCode: 1,
			}, nil
		},
		InspectImpl: func(_ context.Context) (client.ContainerInspectResult, error) {
			return client.ContainerInspectResult{
				Container: container.InspectResponse{
					NetworkSettings: &container.NetworkSettings{
						Ports: network.PortMap{
							network.MustParsePort("8080/tcp"): []network.PortBinding{
								{
									HostIP:   netip.MustParseAddr("127.0.0.1"),
									HostPort: "49152",
								},
							},
						},
					},
				},
			}, nil
		},
	}

	wg := wait.ForHTTP("/").
		WithTimeout(500 * time.Millisecond).
		WithPollInterval(100 * time.Millisecond)

	err := wg.WaitUntilReady(context.Background(), target)
	expected := "container exited with code 1"
	require.EqualError(t, err, expected)
}

func TestHttpStrategyFailsWhileGettingPortDueToUnexpectedContainerStatus(t *testing.T) {
	var mappedPortCount int
	target := &wait.MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			defer func() { mappedPortCount++ }()
			if mappedPortCount == 0 {
				return network.Port{}, wait.ErrPortNotFound
			}
			return network.MustParsePort("49152"), nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Status: "dead",
			}, nil
		},
		InspectImpl: func(_ context.Context) (client.ContainerInspectResult, error) {
			return client.ContainerInspectResult{
				Container: container.InspectResponse{
					NetworkSettings: &container.NetworkSettings{
						Ports: network.PortMap{
							network.MustParsePort("8080/tcp"): []network.PortBinding{
								{
									HostIP:   netip.MustParseAddr("127.0.0.1"),
									HostPort: "49152",
								},
							},
						},
					},
				},
			}, nil
		},
	}

	wg := wait.ForHTTP("/").
		WithTimeout(500 * time.Millisecond).
		WithPollInterval(100 * time.Millisecond)

	err := wg.WaitUntilReady(context.Background(), target)
	expected := "unexpected container status \"dead\""
	require.EqualError(t, err, expected)
}

func TestHTTPStrategyFailsWhileRequestSendingDueToOOMKilledContainer(t *testing.T) {
	target := &wait.MockStrategyTarget{
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
		InspectImpl: func(_ context.Context) (client.ContainerInspectResult, error) {
			return client.ContainerInspectResult{
				Container: container.InspectResponse{
					NetworkSettings: &container.NetworkSettings{
						Ports: network.PortMap{
							network.MustParsePort("8080/tcp"): []network.PortBinding{
								{
									HostIP:   netip.MustParseAddr("127.0.0.1"),
									HostPort: "49152",
								},
							},
						},
					},
				},
			}, nil
		},
	}

	wg := wait.ForHTTP("/").
		WithTimeout(500 * time.Millisecond).
		WithPollInterval(100 * time.Millisecond)

	err := wg.WaitUntilReady(context.Background(), target)
	expected := "container crashed with out-of-memory (OOMKilled)"
	require.EqualError(t, err, expected)
}

func TestHttpStrategyFailsWhileRequestSendingDueToExitedContainer(t *testing.T) {
	target := &wait.MockStrategyTarget{
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
		InspectImpl: func(_ context.Context) (client.ContainerInspectResult, error) {
			return client.ContainerInspectResult{
				Container: container.InspectResponse{
					NetworkSettings: &container.NetworkSettings{
						Ports: network.PortMap{
							network.MustParsePort("8080/tcp"): []network.PortBinding{
								{
									HostIP:   netip.MustParseAddr("127.0.0.1"),
									HostPort: "49152",
								},
							},
						},
					},
				},
			}, nil
		},
	}

	wg := wait.ForHTTP("/").
		WithTimeout(500 * time.Millisecond).
		WithPollInterval(100 * time.Millisecond)

	err := wg.WaitUntilReady(context.Background(), target)
	expected := "container exited with code 1"
	require.EqualError(t, err, expected)
}

func TestHttpStrategyFailsWhileRequestSendingDueToUnexpectedContainerStatus(t *testing.T) {
	target := &wait.MockStrategyTarget{
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
		InspectImpl: func(_ context.Context) (client.ContainerInspectResult, error) {
			return client.ContainerInspectResult{
				Container: container.InspectResponse{
					NetworkSettings: &container.NetworkSettings{
						Ports: network.PortMap{
							network.MustParsePort("8080/tcp"): []network.PortBinding{
								{
									HostIP:   netip.MustParseAddr("127.0.0.1"),
									HostPort: "49152",
								},
							},
						},
					},
				},
			}, nil
		},
	}

	wg := wait.ForHTTP("/").
		WithTimeout(500 * time.Millisecond).
		WithPollInterval(100 * time.Millisecond)

	err := wg.WaitUntilReady(context.Background(), target)
	expected := "unexpected container status \"dead\""
	require.EqualError(t, err, expected)
}

func TestHttpStrategyFailsWhileGettingPortDueToNoExposedPorts(t *testing.T) {
	var mappedPortCount int
	target := &wait.MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			defer func() { mappedPortCount++ }()
			if mappedPortCount == 0 {
				return network.Port{}, wait.ErrPortNotFound
			}
			return network.MustParsePort("49152"), nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Status:  "running",
				Running: true,
			}, nil
		},
		InspectImpl: func(_ context.Context) (client.ContainerInspectResult, error) {
			return client.ContainerInspectResult{
				Container: container.InspectResponse{
					NetworkSettings: &container.NetworkSettings{
						Ports: network.PortMap{},
					},
				},
			}, nil
		},
	}

	wg := wait.ForHTTP("/").
		WithTimeout(500 * time.Millisecond).
		WithPollInterval(100 * time.Millisecond)

	err := wg.WaitUntilReady(context.Background(), target)
	expected := "no exposed tcp ports or mapped ports - cannot wait for status"
	require.EqualError(t, err, expected)
}

func TestHttpStrategyFailsWhileGettingPortDueToOnlyUDPPorts(t *testing.T) {
	var mappedPortCount int
	target := &wait.MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			defer func() { mappedPortCount++ }()
			if mappedPortCount == 0 {
				return network.Port{}, wait.ErrPortNotFound
			}
			return network.MustParsePort("49152"), nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Running: true,
				Status:  "running",
			}, nil
		},
		InspectImpl: func(_ context.Context) (client.ContainerInspectResult, error) {
			return client.ContainerInspectResult{
				Container: container.InspectResponse{
					NetworkSettings: &container.NetworkSettings{
						Ports: network.PortMap{
							network.MustParsePort("8080/udp"): []network.PortBinding{
								{
									HostIP:   netip.MustParseAddr("127.0.0.1"),
									HostPort: "49152",
								},
							},
						},
					},
				},
			}, nil
		},
	}

	wg := wait.ForHTTP("/").
		WithTimeout(500 * time.Millisecond).
		WithPollInterval(100 * time.Millisecond)

	err := wg.WaitUntilReady(context.Background(), target)
	expected := "no exposed tcp ports or mapped ports - cannot wait for status"
	require.EqualError(t, err, expected)
}

func TestHttpStrategyFailsWhileGettingPortDueToExposedPortNoBindings(t *testing.T) {
	var mappedPortCount int
	target := &wait.MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, _ network.Port) (network.Port, error) {
			defer func() { mappedPortCount++ }()
			if mappedPortCount == 0 {
				return network.Port{}, wait.ErrPortNotFound
			}
			return network.MustParsePort("49152"), nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Running: true,
				Status:  "running",
			}, nil
		},
		InspectImpl: func(_ context.Context) (client.ContainerInspectResult, error) {
			return client.ContainerInspectResult{
				Container: container.InspectResponse{
					NetworkSettings: &container.NetworkSettings{
						Ports: network.PortMap{
							network.MustParsePort("8080/tcp"): []network.PortBinding{},
						},
					},
				},
			}, nil
		},
	}

	wg := wait.ForHTTP("/").
		WithTimeout(500 * time.Millisecond).
		WithPollInterval(100 * time.Millisecond)

	err := wg.WaitUntilReady(context.Background(), target)
	expected := "no exposed tcp ports or mapped ports - cannot wait for status"
	require.EqualError(t, err, expected)
}
