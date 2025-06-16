package client_test

import (
	"errors"
	"testing"

	"github.com/containerd/errdefs"
	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/client"
)

func TestIsPermanentClientError(t *testing.T) {
	t.Run("permanent-client-errors", func(t *testing.T) {
		require.True(t, client.IsPermanentClientError(errdefs.ErrNotFound))
		require.True(t, client.IsPermanentClientError(errdefs.ErrInvalidArgument))
		require.True(t, client.IsPermanentClientError(errdefs.ErrUnauthenticated))
		require.True(t, client.IsPermanentClientError(errdefs.ErrPermissionDenied))
		require.True(t, client.IsPermanentClientError(errdefs.ErrNotImplemented))
		require.True(t, client.IsPermanentClientError(errdefs.ErrInternal))
	})

	t.Run("non-permanent-client-errors", func(t *testing.T) {
		require.False(t, client.IsPermanentClientError(errors.New("test")))
	})
}
