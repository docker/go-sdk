package dockercontainer_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/dockercontainer"
)

func TestCreateContainer(t *testing.T) {
	ctr, err := dockercontainer.Create(context.Background(), &dockercontainer.Definition{
		Image: "nginx:alpine",
	})
	dockercontainer.CleanupContainer(t, *ctr)
	require.NoError(t, err)
	require.NotNil(t, ctr)
}
