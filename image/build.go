package image

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/moby/go-archive"
	"github.com/moby/go-archive/compression"
	"github.com/moby/term"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/go-sdk/client"
)

// ArchiveBuildContext creates a TAR archive reader from a directory.
// It returns an error if the directory cannot be read or if the files cannot be read.
// This function is useful for creating a build context to build an image.
// The dockerfile path needs to be relative to the build context.
func ArchiveBuildContext(dir string, dockerfile string) (r io.ReadCloser, err error) {
	// always pass context as absolute path
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("absolute path: %w", err)
	}

	dockerIgnoreExists, excluded, err := ParseDockerIgnore(abs)
	if err != nil {
		return nil, fmt.Errorf("parse docker ignore: %w", err)
	}

	includes := []string{".", dockerfile}
	if dockerIgnoreExists {
		// only add .dockerignore if it exists
		includes = append(includes, ".dockerignore")
	}

	buildContext, err := archive.TarWithOptions(
		abs,
		&archive.TarOptions{
			ExcludePatterns: excluded,
			IncludeFiles:    includes,
			Compression:     compression.Gzip,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("tar with options: %w", err)
	}

	return buildContext, nil
}

// ImageBuildClient is a client that can build images.
type ImageBuildClient interface {
	ImageClient

	// ImageBuild builds an image from a build context and options.
	ImageBuild(ctx context.Context, options build.ImageBuildOptions) (build.ImageBuildResponse, error)
}

// BuildFromDir builds an image from a directory and the path to the Dockerfile in the directory, then returns the tag.
// It uses [ArchiveBuildContext] to create a archive reader from the directory.
func BuildFromDir(ctx context.Context, dir string, dockerfile string, tag string, opts ...BuildOption) (string, error) {
	contextArchive, err := ArchiveBuildContext(dir, dockerfile)
	if err != nil {
		return "", fmt.Errorf("archive build context: %w", err)
	}

	buildOpts := build.ImageBuildOptions{
		Dockerfile: dockerfile,
	}

	opts = append(opts, WithBuildOptions(buildOpts))

	return Build(ctx, contextArchive, tag, opts...)
}

// Build will build and image from context and Dockerfile, then return the tag. It uses "Dockerfile" as the Dockerfile path,
// although it can be overridden by the build options.
// In the case the build options contains tags or a context reader, they will be overridden by the arguments passed to the function,
// which are mandatory.
func Build(ctx context.Context, contextReader io.Reader, tag string, opts ...BuildOption) (string, error) {
	// validations happen first to avoid unnecessary allocations
	if contextReader == nil {
		return "", errors.New("context reader is required")
	}

	buildOpts := &buildOptions{
		opts: build.ImageBuildOptions{
			Dockerfile: "Dockerfile",
		},
	}
	for _, opt := range opts {
		if err := opt(buildOpts); err != nil {
			return "", fmt.Errorf("apply build option: %w", err)
		}
	}

	if len(buildOpts.opts.Tags) == 0 {
		buildOpts.opts.Tags = make([]string, 1)
	}

	if tag == "" {
		if len(buildOpts.opts.Tags) == 0 || buildOpts.opts.Tags[0] == "" {
			return "", errors.New("tag cannot be empty")
		}
	}
	// Set the passed tag, even if it is set in the build options.
	buildOpts.opts.Tags[0] = tag

	// Set the passed context reader, even if it is set in the build options.
	buildOpts.opts.Context = contextReader

	if buildOpts.buildClient == nil {
		buildOpts.buildClient = client.DefaultClient
		// In case there is no build client set, use the default docker client
		// to build the image. Needs to be closed when done.
		defer buildOpts.buildClient.Close()
	}

	if buildOpts.opts.Labels == nil {
		buildOpts.opts.Labels = make(map[string]string)
	}

	// Add client labels
	client.AddSDKLabels(buildOpts.opts.Labels)

	// Close the context reader after all retries are complete
	defer tryClose(contextReader)

	resp, err := backoff.RetryNotifyWithData(
		func() (build.ImageBuildResponse, error) {
			var err error

			resp, err := buildOpts.buildClient.ImageBuild(ctx, buildOpts.opts)
			if err != nil {
				if client.IsPermanentClientError(err) {
					return build.ImageBuildResponse{}, backoff.Permanent(fmt.Errorf("build image: %w", err))
				}
				return build.ImageBuildResponse{}, err
			}

			return resp, nil
		},
		backoff.WithContext(backoff.NewExponentialBackOff(), ctx),
		func(err error, _ time.Duration) {
			buildOpts.buildClient.Logger().Warn("Failed to build image, will retry", "error", err)
		},
	)
	if err != nil {
		return "", err // Error is already wrapped.
	}
	defer resp.Body.Close()

	// use the bridge to log to the client logger
	output := &loggerWriter{logger: buildOpts.buildClient.Logger()}

	// Always process the output, even if it is not printed
	// to ensure that errors during the build process are
	// correctly handled.
	termFd, isTerm := term.GetFdInfo(output)
	if err = jsonmessage.DisplayJSONMessagesStream(resp.Body, output, termFd, isTerm, nil); err != nil {
		return "", fmt.Errorf("build image: %w", err)
	}

	// the first tag is the one we want, which must be the passed tag
	return buildOpts.opts.Tags[0], nil
}

func tryClose(r io.Reader) {
	rc, ok := r.(io.Closer)
	if ok {
		_ = rc.Close()
	}
}
