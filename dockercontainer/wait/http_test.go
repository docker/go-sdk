package wait_test

import (
	"context"
	_ "embed"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-sdk/dockercontainer/wait"
)

func TestHttpStrategyFailsWhileGettingPortDueToOOMKilledContainer(t *testing.T) {
	var mappedPortCount int
	target := &wait.MockStrategyTarget{
		HostImpl: func(_ context.Context) (string, error) {
			return "localhost", nil
		},
		MappedPortImpl: func(_ context.Context, _ nat.Port) (nat.Port, error) {
			defer func() { mappedPortCount++ }()
			if mappedPortCount == 0 {
				return "", wait.ErrPortNotFound
			}
			return "49152", nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				OOMKilled: true,
			}, nil
		},
		InspectImpl: func(_ context.Context) (*container.InspectResponse, error) {
			return &container.InspectResponse{
				NetworkSettings: &container.NetworkSettings{
					NetworkSettingsBase: container.NetworkSettingsBase{
						Ports: nat.PortMap{
							"8080/tcp": []nat.PortBinding{
								{
									HostIP:   "127.0.0.1",
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
		MappedPortImpl: func(_ context.Context, _ nat.Port) (nat.Port, error) {
			defer func() { mappedPortCount++ }()
			if mappedPortCount == 0 {
				return "", wait.ErrPortNotFound
			}
			return "49152", nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Status:   "exited",
				ExitCode: 1,
			}, nil
		},
		InspectImpl: func(_ context.Context) (*container.InspectResponse, error) {
			return &container.InspectResponse{
				NetworkSettings: &container.NetworkSettings{
					NetworkSettingsBase: container.NetworkSettingsBase{
						Ports: nat.PortMap{
							"8080/tcp": []nat.PortBinding{
								{
									HostIP:   "127.0.0.1",
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
		MappedPortImpl: func(_ context.Context, _ nat.Port) (nat.Port, error) {
			defer func() { mappedPortCount++ }()
			if mappedPortCount == 0 {
				return "", wait.ErrPortNotFound
			}
			return "49152", nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Status: "dead",
			}, nil
		},
		InspectImpl: func(_ context.Context) (*container.InspectResponse, error) {
			return &container.InspectResponse{
				NetworkSettings: &container.NetworkSettings{
					NetworkSettingsBase: container.NetworkSettingsBase{
						Ports: nat.PortMap{
							"8080/tcp": []nat.PortBinding{
								{
									HostIP:   "127.0.0.1",
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
		MappedPortImpl: func(_ context.Context, _ nat.Port) (nat.Port, error) {
			return "49152", nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				OOMKilled: true,
			}, nil
		},
		InspectImpl: func(_ context.Context) (*container.InspectResponse, error) {
			return &container.InspectResponse{
				NetworkSettings: &container.NetworkSettings{
					NetworkSettingsBase: container.NetworkSettingsBase{
						Ports: nat.PortMap{
							"8080/tcp": []nat.PortBinding{
								{
									HostIP:   "127.0.0.1",
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
		MappedPortImpl: func(_ context.Context, _ nat.Port) (nat.Port, error) {
			return "49152", nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Status:   "exited",
				ExitCode: 1,
			}, nil
		},
		InspectImpl: func(_ context.Context) (*container.InspectResponse, error) {
			return &container.InspectResponse{
				NetworkSettings: &container.NetworkSettings{
					NetworkSettingsBase: container.NetworkSettingsBase{
						Ports: nat.PortMap{
							"8080/tcp": []nat.PortBinding{
								{
									HostIP:   "127.0.0.1",
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
		MappedPortImpl: func(_ context.Context, _ nat.Port) (nat.Port, error) {
			return "49152", nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Status: "dead",
			}, nil
		},
		InspectImpl: func(_ context.Context) (*container.InspectResponse, error) {
			return &container.InspectResponse{
				NetworkSettings: &container.NetworkSettings{
					NetworkSettingsBase: container.NetworkSettingsBase{
						Ports: nat.PortMap{
							"8080/tcp": []nat.PortBinding{
								{
									HostIP:   "127.0.0.1",
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
		MappedPortImpl: func(_ context.Context, _ nat.Port) (nat.Port, error) {
			defer func() { mappedPortCount++ }()
			if mappedPortCount == 0 {
				return "", wait.ErrPortNotFound
			}
			return "49152", nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Status:  "running",
				Running: true,
			}, nil
		},
		InspectImpl: func(_ context.Context) (*container.InspectResponse, error) {
			return &container.InspectResponse{
				NetworkSettings: &container.NetworkSettings{
					NetworkSettingsBase: container.NetworkSettingsBase{
						Ports: nat.PortMap{},
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
		MappedPortImpl: func(_ context.Context, _ nat.Port) (nat.Port, error) {
			defer func() { mappedPortCount++ }()
			if mappedPortCount == 0 {
				return "", wait.ErrPortNotFound
			}
			return "49152", nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Running: true,
				Status:  "running",
			}, nil
		},
		InspectImpl: func(_ context.Context) (*container.InspectResponse, error) {
			return &container.InspectResponse{
				NetworkSettings: &container.NetworkSettings{
					NetworkSettingsBase: container.NetworkSettingsBase{
						Ports: nat.PortMap{
							"8080/udp": []nat.PortBinding{
								{
									HostIP:   "127.0.0.1",
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
		MappedPortImpl: func(_ context.Context, _ nat.Port) (nat.Port, error) {
			defer func() { mappedPortCount++ }()
			if mappedPortCount == 0 {
				return "", wait.ErrPortNotFound
			}
			return "49152", nil
		},
		StateImpl: func(_ context.Context) (*container.State, error) {
			return &container.State{
				Running: true,
				Status:  "running",
			}, nil
		},
		InspectImpl: func(_ context.Context) (*container.InspectResponse, error) {
			return &container.InspectResponse{
				NetworkSettings: &container.NetworkSettings{
					NetworkSettingsBase: container.NetworkSettingsBase{
						Ports: nat.PortMap{
							"8080/tcp": []nat.PortBinding{},
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
