package image

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/pkg/authconfig"
	dockerclient "github.com/moby/moby/client"
	"github.com/stretchr/testify/require"

	sdkclient "github.com/docker/go-sdk/client"
)

func TestPullRegistryAuth(t *testing.T) {
	mockCli := &errMockCli{}
	sdk, err := sdkclient.New(context.TODO(), sdkclient.WithDockerAPI(mockCli))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	imageName := "myregistry.example.com/myimage:tag"
	err = Pull(ctx, imageName,
		WithPullOptions(dockerclient.ImagePullOptions{}),
		WithCredentialsFn(func(_ string) (string, string, error) {
			return "user", "pass", nil
		}),
		WithPullClient(sdk),
	)
	require.NoError(t, err)

	// Decode the RegistryAuth header and verify ServerAddress is set from the image name.
	decoded, err := authconfig.Decode(mockCli.lastPullOptions.RegistryAuth)
	require.NoError(t, err)
	require.Equal(t, "user", decoded.Username)
	require.Equal(t, "pass", decoded.Password)
	require.Equal(t, "myregistry.example.com", decoded.ServerAddress)
}

func TestPullRegistryAuth_TokenUsername(t *testing.T) {
	mockCli := &errMockCli{}
	sdk, err := sdkclient.New(context.TODO(), sdkclient.WithDockerAPI(mockCli))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	imageName := "docker/sandbox-templates:shell-docker"
	err = Pull(ctx, imageName,
		WithPullOptions(dockerclient.ImagePullOptions{}),
		WithCredentialsFn(func(_ string) (string, string, error) {
			return "<token>", "my-oauth-token", nil
		}),
		WithPullClient(sdk),
	)
	require.NoError(t, err)

	// When username is "<token>", the token should be mapped to IdentityToken
	// (not Username/Password), matching the Docker CLI credential store convention.
	decoded, err := authconfig.Decode(mockCli.lastPullOptions.RegistryAuth)
	require.NoError(t, err)
	require.Empty(t, decoded.Username, "Username should be empty for token auth")
	require.Empty(t, decoded.Password, "Password should be empty for token auth")
	require.Equal(t, "my-oauth-token", decoded.IdentityToken)
	require.Equal(t, "docker.io", decoded.ServerAddress)
}

func TestPull(t *testing.T) {
	defaultPullOpts := []PullOption{WithPullOptions(dockerclient.ImagePullOptions{})}

	testPull := func(t *testing.T, imageName string, pullOpts []PullOption, mockCli *errMockCli, shouldRetry bool) string {
		t.Helper()
		buf := &bytes.Buffer{}

		if len(pullOpts) > 0 && mockCli != nil {
			sdk, err := sdkclient.New(context.TODO(),
				sdkclient.WithDockerAPI(mockCli),
				sdkclient.WithLogger(slog.New(slog.NewTextHandler(buf, nil))))
			require.NoError(t, err)

			pullOpts = append(pullOpts, WithPullClient(sdk))
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		err := Pull(ctx, imageName, pullOpts...)
		if mockCli.err != nil {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		defer mockCli.Close()

		// Only validate the retry logic if there are more than 1 pull option.
		if len(pullOpts) > 1 {
			require.Positive(t, mockCli.imagePullCount)
			require.Equal(t, shouldRetry, mockCli.imagePullCount > 1)
		}
		return buf.String()
	}

	t.Run("error/no-image", func(t *testing.T) {
		testPull(t, "", []PullOption{}, &errMockCli{err: errors.New("image name is not set")}, false)
	})

	t.Run("error/no-client", func(t *testing.T) {
		testPull(t, "someTag", []PullOption{}, &errMockCli{err: errors.New("image name is not set")}, false)
	})

	t.Run("success/no-retry", func(t *testing.T) {
		testPull(t, "some_tag", defaultPullOpts, &errMockCli{err: nil}, false)
	})

	t.Run("not-available/no-retry", func(t *testing.T) {
		testPull(t, "some_tag", defaultPullOpts, &errMockCli{err: errdefs.ErrNotFound.WithMessage("not available")}, false)
	})

	t.Run("invalid-parameters/no-retry", func(t *testing.T) {
		testPull(t, "some_tag", defaultPullOpts, &errMockCli{err: errdefs.ErrInvalidArgument.WithMessage("invalid")}, false)
	})

	t.Run("unauthorized/retry", func(t *testing.T) {
		testPull(t, "some_tag", defaultPullOpts, &errMockCli{err: errdefs.ErrUnauthenticated.WithMessage("not authorized")}, false)
	})

	t.Run("forbidden/retry", func(t *testing.T) {
		testPull(t, "some_tag", defaultPullOpts, &errMockCli{err: errdefs.ErrPermissionDenied.WithMessage("forbidden")}, false)
	})

	t.Run("not-implemented/retry", func(t *testing.T) {
		testPull(t, "some_tag", defaultPullOpts, &errMockCli{err: errdefs.ErrNotImplemented.WithMessage("unknown method")}, false)
	})

	t.Run("non-permanent-error/retry", func(t *testing.T) {
		mockCliWithLogger := &errMockCli{
			err: errors.New("whoops"),
		}

		out := testPull(t, "some_tag", defaultPullOpts, mockCliWithLogger, true)
		require.Contains(t, out, "failed to pull image, will retry")
	})
}

func TestPullDefaultCredentials(t *testing.T) {
	newSDK := func(t *testing.T, mockCli *errMockCli, buf *bytes.Buffer) sdkclient.SDKClient {
		t.Helper()
		sdk, err := sdkclient.New(context.TODO(),
			sdkclient.WithDockerAPI(mockCli),
			sdkclient.WithLogger(slog.New(slog.NewTextHandler(buf, nil))))
		require.NoError(t, err)
		return sdk
	}

	t.Run("broken-config/anonymous-fallback", func(t *testing.T) {
		tmpDir := t.TempDir()
		// unique helper name to avoid the config module's content-hash cache
		helper := fmt.Sprintf("broken-helper-%d", time.Now().UnixNano())
		binDir := t.TempDir()
		helperPath := filepath.Join(binDir, "docker-credential-"+helper)
		require.NoError(t, os.WriteFile(helperPath, []byte("#!/bin/sh\nexit 1\n"), 0o755))
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		cfg := fmt.Sprintf(`{"credHelpers":{"myregistry.example.com":%q}}`, helper)
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "config.json"), []byte(cfg), 0o600))
		t.Setenv("DOCKER_CONFIG", tmpDir)
		t.Setenv("DOCKER_AUTH_CONFIG", "")

		mockCli := &errMockCli{}
		buf := &bytes.Buffer{}
		sdk := newSDK(t, mockCli, buf)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		err := Pull(ctx, "myregistry.example.com/myimage:tag",
			WithPullOptions(dockerclient.ImagePullOptions{}),
			WithPullClient(sdk),
		)
		require.NoError(t, err)
		require.Contains(t, buf.String(), "failed to retrieve registry credentials, pulling without authentication")
		require.Positive(t, mockCli.imagePullCount)

		decoded, err := authconfig.Decode(mockCli.lastPullOptions.RegistryAuth)
		require.NoError(t, err)
		require.Empty(t, decoded.Username)
		require.Empty(t, decoded.Password)
		require.Empty(t, decoded.IdentityToken)
	})

	t.Run("no-config/anonymous", func(t *testing.T) {
		t.Setenv("DOCKER_CONFIG", t.TempDir())
		t.Setenv("DOCKER_AUTH_CONFIG", "")

		mockCli := &errMockCli{}
		buf := &bytes.Buffer{}
		sdk := newSDK(t, mockCli, buf)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		err := Pull(ctx, "myregistry.example.com/myimage:tag",
			WithPullOptions(dockerclient.ImagePullOptions{}),
			WithPullClient(sdk),
		)
		require.NoError(t, err)
		require.Positive(t, mockCli.imagePullCount)

		decoded, err := authconfig.Decode(mockCli.lastPullOptions.RegistryAuth)
		require.NoError(t, err)
		require.Empty(t, decoded.Username)
		require.Empty(t, decoded.Password)
		require.Empty(t, decoded.IdentityToken)
	})

	t.Run("explicit-credentials-fn/error-fails", func(t *testing.T) {
		mockCli := &errMockCli{}
		buf := &bytes.Buffer{}
		sdk := newSDK(t, mockCli, buf)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		err := Pull(ctx, "myregistry.example.com/myimage:tag",
			WithPullOptions(dockerclient.ImagePullOptions{}),
			WithCredentialsFn(func(string) (string, string, error) {
				return "", "", errors.New("boom")
			}),
			WithPullClient(sdk),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to retrieve registry credentials")
		require.Zero(t, mockCli.imagePullCount)
	})
}
