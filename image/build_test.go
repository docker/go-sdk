package image_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/docker/go-sdk/client"
	"github.com/docker/go-sdk/image"
)

const (
	labelImageBuildTestKey   = "image.build.test"
	labelImageBuildTestValue = "true"
)

// Used as a marker to identify the containers created by the test
// so it's possible to clean them up after the tests.
var labelImageBuildTest = map[string]string{
	labelImageBuildTestKey: labelImageBuildTestValue,
}

type testBuildInfo struct {
	imageTag       string
	buildErr       error
	contextArchive io.ReadSeeker
	logWriter      io.Writer
}

func TestBuild(t *testing.T) {
	files := []image.BuildFile{
		{
			Name:    "say_hi.sh",
			Content: []byte(`echo hi this is from the say_hi.sh file!`),
		},
		{
			Name: "Dockerfile",
			Content: []byte(`FROM alpine
					WORKDIR /app
					COPY . .
					CMD ["sh", "./say_hi.sh"]`),
		},
	}

	t.Run("success", func(t *testing.T) {
		contextArchive, err := image.ReaderFromFiles(files)
		require.NoError(t, err)

		b := &testBuildInfo{
			contextArchive: contextArchive,
			logWriter:      &bytes.Buffer{},
			imageTag:       "test:test",
		}
		testBuild(t, b)
	})

	t.Run("success/with-client", func(t *testing.T) {
		contextArchive, err := image.ReaderFromFiles(files)
		require.NoError(t, err)

		b := &testBuildInfo{
			contextArchive: contextArchive,
			logWriter:      &bytes.Buffer{},
			imageTag:       "test:test",
		}

		cli, err := client.New(context.Background())
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, cli.Close())
		})

		testBuild(t, b, image.WithBuildClient(cli))
	})

	t.Run("error/image-tag-empty", func(t *testing.T) {
		b := &testBuildInfo{
			buildErr: errors.New("tag cannot be empty"),
		}

		testBuild(t, b)
	})

	t.Run("error/context-reader-nil", func(t *testing.T) {
		b := &testBuildInfo{
			imageTag: "test:test",
			buildErr: errors.New("context reader is required"),
		}

		testBuild(t, b)
	})
}

func testBuild(tb testing.TB, b *testBuildInfo, opts ...image.BuildOption) {
	tb.Helper()

	cli, err := client.New(context.Background())
	require.NoError(tb, err)
	tb.Cleanup(func() {
		require.NoError(tb, cli.Close())
	})

	opts = append(opts, image.WithBuildOptions(build.ImageBuildOptions{
		Labels: labelImageBuildTest,
	}))

	tag, err := image.Build(context.Background(), b.contextArchive, b.imageTag, opts...)

	if b.buildErr != nil {
		// build error is the error returned by the build
		require.ErrorContains(tb, err, b.buildErr.Error())
		require.Empty(tb, tag)

		return
	}

	defer func() {
		_, err = image.Remove(context.Background(), tag, image.WithRemoveOptions(dockerimage.RemoveOptions{
			Force:         true,
			PruneChildren: true,
		}))
		require.NoError(tb, err)

		containers, err := cli.ContainerList(context.Background(), container.ListOptions{
			Filters: filters.NewArgs(filters.Arg("status", "created"), filters.Arg("label", fmt.Sprintf("%s=%s", client.LabelBase, "true"))),
			All:     true,
		})
		require.NoError(tb, err)

		// force the removal of the intermediate containers, if any
		for _, ctr := range containers {
			require.NoError(tb, cli.ContainerRemove(context.Background(), ctr.ID, container.RemoveOptions{Force: true}))
		}
	}()

	require.NoError(tb, err)
	require.Equal(tb, b.imageTag, tag)
}
