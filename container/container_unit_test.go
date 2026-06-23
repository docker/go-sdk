package container

import (
	"context"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/stretchr/testify/require"
)

func TestFromResponse(t *testing.T) {
	response := container.Summary{
		ID:    "1234567890abcdefgh",
		Image: "nginx:latest",
		State: "running",
		Ports: []container.PortSummary{
			{PublicPort: 80, Type: "tcp"},
			{PublicPort: 8080, Type: "udp"},
		},
		Names: []string{"/my-container", "/other-name"},
	}

	ctr, err := FromResponse(context.Background(), nil, response)
	require.NoError(t, err)
	require.Equal(t, "1234567890abcdefgh", ctr.ID())
	require.Equal(t, "1234567890ab", ctr.ShortID())
	require.Equal(t, "nginx:latest", ctr.Image())
	require.Equal(t, "/my-container", ctr.Name())
	require.Equal(t, []string{"80/tcp", "8080/udp"}, ctr.exposedPorts)
	require.NotNil(t, ctr.dockerClient)
	require.NotNil(t, ctr.logger)
}
