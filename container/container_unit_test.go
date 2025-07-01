package container

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/container"
)

func TestFromResponse(t *testing.T) {
	response := container.Summary{
		ID:    "1234567890abcdefgh",
		Image: "nginx:latest",
		State: "running",
		Ports: []container.Port{
			{PublicPort: 80, Type: "tcp"},
			{PublicPort: 8080, Type: "udp"},
		},
	}

	ctr, err := FromResponse(context.Background(), response)
	require.NoError(t, err)
	require.Equal(t, "1234567890abcdefgh", ctr.ID())
	require.Equal(t, "1234567890ab", ctr.ShortID())
	require.Equal(t, "nginx:latest", ctr.Image())
	require.Equal(t, []string{"80/tcp", "8080/udp"}, ctr.exposedPorts)
}
