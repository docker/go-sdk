package container_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/container"
)

func TestFromID(t *testing.T) {
	// First, create a container using Run
	ctr, err := container.Run(context.Background(), container.WithImage("alpine:latest"))
	require.NoError(t, err)

	// Use the SDK client from the existing container
	cli := ctr.Client()

	// Now recreate the container using FromID with the container ID
	// This is useful when you only have a container ID and need to perform operations on it
	recreated, err := container.FromID(context.Background(), cli, ctr.ID())
	require.NoError(t, err)
	require.Equal(t, ctr.ID(), recreated.ID())

	// Verify operations like CopyToContainer on the recreated container
	content := []byte("Hello from FromID!")
	require.NoError(t, recreated.CopyToContainer(context.Background(), content, "/tmp/test.txt", 0o644))

	rc, err := recreated.CopyFromContainer(context.Background(), "/tmp/test.txt")
	require.NoError(t, err)

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, rc)
	require.NoError(t, err)
	require.Equal(t, string(content), buf.String())

	// Terminate the recreated container
	err = recreated.Terminate(context.Background())
	require.NoError(t, err)

	// Terminate the original container should fail
	err = ctr.Terminate(context.Background())
	require.Error(t, err)
}
