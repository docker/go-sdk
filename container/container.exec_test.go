package container_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/go-sdk/container"
	"github.com/docker/go-sdk/container/exec"
)

const (
	alpineLatest     = "alpine:latest"
	nginxAlpineImage = "nginx:alpine"
	bashImage        = "bash:5.2.26"
)

func TestContainer_Exec(t *testing.T) {
	t.Run("stopped-container/error", func(t *testing.T) {
		ctr, err := container.Run(context.Background(),
			container.WithImage(alpineLatest),
			container.WithNoStart(),
		)
		require.NoError(t, err)

		container.Cleanup(t, ctr)

		_, reader, err := ctr.Exec(context.Background(), []string{"ls", "-l"})
		require.Error(t, err)
		require.Nil(t, reader)
	})

	t.Run("running-container", func(t *testing.T) {
		ctr, err := container.Run(context.Background(),
			// using an image that has a long-running command
			container.WithImage(nginxAlpineImage),
		)
		require.NoError(t, err)

		container.Cleanup(t, ctr)

		t.Run("success", func(t *testing.T) {
			code, reader, err := ctr.Exec(context.Background(), []string{"ls", "-l"})
			require.NoError(t, err)
			require.NotNil(t, reader)
			require.Equal(t, 0, code)
		})

		t.Run("error", func(t *testing.T) {
			code, reader, err := ctr.Exec(context.Background(), []string{"non-existent-command"})
			require.NoError(t, err)
			require.NotNil(t, reader)
			require.Equal(t, 127, code)
		})

		t.Run("with-multiplexed-reader", func(t *testing.T) {
			code, reader, err := ctr.Exec(context.Background(), []string{"ls", "-l"}, exec.Multiplexed())
			require.NoError(t, err)
			require.NotNil(t, reader)
			require.Equal(t, 0, code)
		})

		t.Run("with-user", func(t *testing.T) {
			code, reader, err := ctr.Exec(context.Background(), []string{"ls", "-l"}, exec.WithUser("root"))
			require.NoError(t, err)
			require.NotNil(t, reader)
			require.Equal(t, 0, code)
		})

		t.Run("with-working-dir", func(t *testing.T) {
			code, reader, err := ctr.Exec(context.Background(), []string{"pwd"}, exec.WithWorkingDir("/tmp"), exec.Multiplexed())
			require.NoError(t, err)
			require.NotNil(t, reader)
			require.Equal(t, 0, code)

			buf := &bytes.Buffer{}
			_, err = io.Copy(buf, reader)
			require.NoError(t, err)
			require.Equal(t, "/tmp\n", buf.String())
		})

		t.Run("with-env", func(t *testing.T) {
			code, reader, err := ctr.Exec(context.Background(), []string{"printenv"}, exec.WithEnv([]string{"FOO=bar"}), exec.Multiplexed())
			require.NoError(t, err)
			require.NotNil(t, reader)
			require.Equal(t, 0, code)

			buf := &bytes.Buffer{}
			_, err = io.Copy(buf, reader)
			require.NoError(t, err)
			require.Contains(t, buf.String(), "FOO=bar")
		})

		t.Run("with-tty", func(t *testing.T) {
			code, reader, err := ctr.Exec(context.Background(), []string{"ls", "-l"}, exec.WithTTY(true))
			require.NoError(t, err)
			require.NotNil(t, reader)
			require.Equal(t, 0, code)
		})
	})
}
