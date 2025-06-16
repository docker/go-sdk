package image

import (
	"bytes"
	"context"
	"io"
	"log/slog"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

// errMockCli is a mock implementation of client.APIClient, which is handy for simulating
// error returns in retry scenarios.
type errMockCli struct {
	client.APIClient

	err            error
	imagePullCount int
	logger         *slog.Logger
}

func (f *errMockCli) ImageBuild(_ context.Context, _ io.Reader, _ build.ImageBuildOptions) (build.ImageBuildResponse, error) {
	return build.ImageBuildResponse{}, f.err
}

func (f *errMockCli) ImagePull(_ context.Context, _ string, _ image.PullOptions) (io.ReadCloser, error) {
	f.imagePullCount++
	return io.NopCloser(&bytes.Buffer{}), f.err
}

func (f *errMockCli) Close() error {
	return nil
}

func (f *errMockCli) Logger() *slog.Logger {
	return f.logger
}
