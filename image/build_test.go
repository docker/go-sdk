package image_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path"
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

type testBuildInfo struct {
	imageTag       string
	buildErr       error
	contextArchive io.Reader
	logWriter      io.Writer
	dockerfilePath string
}

func TestBuild(t *testing.T) {
	buildPath := path.Join("testdata", "build")

	t.Run("success", func(t *testing.T) {
		contextArchive, err := image.ArchiveBuildContext(buildPath, "Dockerfile")
		require.NoError(t, err)

		b := &testBuildInfo{
			contextArchive: contextArchive,
			logWriter:      &bytes.Buffer{},
			imageTag:       "test:test",
		}
		testBuild(t, b)
	})

	t.Run("success/with-client", func(t *testing.T) {
		contextArchive, err := image.ArchiveBuildContext(buildPath, "Dockerfile")
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
		contextArchive, err := image.ArchiveBuildContext(buildPath, "Dockerfile")
		require.NoError(t, err)

		b := &testBuildInfo{
			contextArchive: contextArchive,
			buildErr:       errors.New("tag cannot be empty"),
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

	t.Run("error/dockerfile-not-found-in-context", func(t *testing.T) {
		contextArchive, err := image.ArchiveBuildContext(buildPath, "Dockerfile")
		require.NoError(t, err)

		b := &testBuildInfo{
			contextArchive: contextArchive,
			logWriter:      &bytes.Buffer{},
			imageTag:       "test:test",
			dockerfilePath: "Dockerfile.not-found",
			buildErr:       errors.New("Cannot locate specified Dockerfile: Dockerfile.not-found"),
		}
		testBuild(t, b)
	})
}

func TestBuildFromDir(t *testing.T) {
	buildPath := path.Join("testdata", "build")

	t.Run("success", func(t *testing.T) {
		tag, err := image.BuildFromDir(context.Background(), buildPath, "Dockerfile", "test:test")
		t.Cleanup(func() {
			cleanup(t, tag)
		})
		require.NoError(t, err)
		require.Equal(t, "test:test", tag)
	})

	t.Run("with-dockerfile/options-are-overridden", func(t *testing.T) {
		tag, err := image.BuildFromDir(context.Background(), buildPath, "Dockerfile", "test:test", image.WithBuildOptions(build.ImageBuildOptions{
			Dockerfile: "Dockerfile.custom",
		}))
		t.Cleanup(func() {
			cleanup(t, tag)
		})
		require.NoError(t, err)
		require.Equal(t, "test:test", tag)
	})
}

func TestBuild_addSDKLabels(t *testing.T) {
	buildPath := path.Join("testdata", "build")

	tag, err := image.BuildFromDir(context.Background(), buildPath, "Dockerfile", "test:test")
	require.NoError(t, err)
	require.Equal(t, "test:test", tag)
	t.Cleanup(func() {
		cleanup(t, tag)
	})

	inspect, err := client.DefaultClient.ImageInspect(context.Background(), tag)
	require.NoError(t, err)

	require.Contains(t, inspect.Config.Labels, client.LabelBase)
	require.Contains(t, inspect.Config.Labels, client.LabelLang)
	require.Contains(t, inspect.Config.Labels, client.LabelVersion)
	require.Contains(t, inspect.Config.Labels, client.LabelBase+".image")
	require.Equal(t, image.Version(), inspect.Config.Labels[client.LabelBase+".image"])
}

func testBuild(tb testing.TB, b *testBuildInfo, opts ...image.BuildOption) {
	tb.Helper()

	cliOpts := []client.ClientOption{}
	if b.logWriter != nil {
		cliOpts = append(cliOpts, client.WithLogger(slog.New(slog.NewTextHandler(b.logWriter, nil))))
	}

	cli, err := client.New(context.Background(), cliOpts...)
	require.NoError(tb, err)
	tb.Cleanup(func() {
		require.NoError(tb, cli.Close())
	})

	buildOpts := build.ImageBuildOptions{
		// Used as a marker to identify the containers created by the test
		// so it's possible to clean them up after the tests.
		Labels: map[string]string{
			labelImageBuildTestKey: labelImageBuildTestValue,
		},
	}
	if b.dockerfilePath != "" {
		buildOpts.Dockerfile = b.dockerfilePath
	}

	opts = append(opts, image.WithBuildOptions(buildOpts))

	tag, err := image.Build(context.Background(), b.contextArchive, b.imageTag, opts...)

	if b.buildErr != nil {
		// build error is the error returned by the build
		require.ErrorContains(tb, err, b.buildErr.Error())
		require.Empty(tb, tag)

		return
	}

	tb.Cleanup(func() {
		cleanup(tb, tag)
	})

	require.NoError(tb, err)
	require.Equal(tb, b.imageTag, tag)
}

func cleanup(tb testing.TB, tag string) {
	tb.Helper()

	cli, err := client.New(context.Background())
	require.NoError(tb, err)
	tb.Cleanup(func() {
		require.NoError(tb, cli.Close())
	})

	_, err = image.Remove(context.Background(), tag, image.WithRemoveOptions(dockerimage.RemoveOptions{
		Force:         true,
		PruneChildren: true,
	}))
	require.NoError(tb, err)

	containers, err := cli.ContainerList(context.Background(), container.ListOptions{
		Filters: filters.NewArgs(filters.Arg("status", "created"), filters.Arg("label", fmt.Sprintf("%s=%s", labelImageBuildTestKey, labelImageBuildTestValue))),
		All:     true,
	})
	require.NoError(tb, err)

	// force the removal of the intermediate containers, if any
	for _, ctr := range containers {
		require.NoError(tb, cli.ContainerRemove(context.Background(), ctr.ID, container.RemoveOptions{Force: true}))
	}
}
