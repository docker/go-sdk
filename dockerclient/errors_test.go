package dockerclient_test

import (
	"errors"
	"testing"

	"github.com/containerd/errdefs"
	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/dockerclient"
)

func TestIsPermanentClientError(t *testing.T) {
	t.Run("permanent-client-errors", func(t *testing.T) {
		require.True(t, dockerclient.IsPermanentClientError(errdefs.ErrNotFound))
		require.True(t, dockerclient.IsPermanentClientError(errdefs.ErrInvalidArgument))
		require.True(t, dockerclient.IsPermanentClientError(errdefs.ErrUnauthenticated))
		require.True(t, dockerclient.IsPermanentClientError(errdefs.ErrPermissionDenied))
		require.True(t, dockerclient.IsPermanentClientError(errdefs.ErrNotImplemented))
		require.True(t, dockerclient.IsPermanentClientError(errdefs.ErrInternal))
	})

	t.Run("non-permanent-client-errors", func(t *testing.T) {
		require.False(t, dockerclient.IsPermanentClientError(errors.New("test")))
	})
}
