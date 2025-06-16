package network

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-sdk/client"
)

// New creates a new network.
func New(ctx context.Context, opts ...Option) (*Network, error) {
	networkOptions := &options{
		labels: make(map[string]string),
	}

	for _, opt := range opts {
		if err := opt(networkOptions); err != nil {
			return nil, fmt.Errorf("apply option: %w", err)
		}
	}

	if networkOptions.name == "" {
		networkOptions.name = uuid.New().String()
	}

	if networkOptions.client == nil {
		dockerClient, err := client.New(context.Background())
		if err != nil {
			return nil, fmt.Errorf("create docker client: %w", err)
		}
		networkOptions.client = dockerClient
	}

	client.AddSDKLabels(networkOptions.labels)

	nc := network.CreateOptions{
		Driver:     networkOptions.driver,
		Internal:   networkOptions.internal,
		EnableIPv6: &networkOptions.enableIPv6,
		Attachable: networkOptions.attachable,
		Labels:     networkOptions.labels,
		IPAM:       networkOptions.ipam,
	}

	resp, err := networkOptions.client.NetworkCreate(ctx, networkOptions.name, nc)
	if err != nil {
		return nil, fmt.Errorf("create network: %w", err)
	}

	if resp.Warning != "" {
		networkOptions.client.Logger().Warn("warning creating network", "message", resp.Warning)
	}

	return &Network{
		response:     resp,
		name:         networkOptions.name,
		opts:         networkOptions,
		dockerClient: networkOptions.client,
	}, nil
}
