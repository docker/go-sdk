package image

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/moby/term"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/go-sdk/client"
)

// BuildFile represents a file that is part of a build context.
type BuildFile struct {
	// Name is the name of the file.
	Name string

	// Content is the content of the file.
	Content []byte
}

// ReaderFromFiles creates a TAR archive reader from a list of files.
// It returns an error if the files cannot be written to the reader.
// This function is useful for creating a build context to build an image.
func ReaderFromFiles(files []BuildFile) (r io.ReadSeeker, err error) {
	var buf bytes.Buffer
	tarWriter := tar.NewWriter(&buf)

	var errs []error
	for _, f := range files {
		err := tarWriter.WriteHeader(&tar.Header{
			Name:     f.Name,
			Mode:     0o777,
			Size:     int64(len(f.Content)),
			Typeflag: tar.TypeReg,
			Format:   tar.FormatGNU,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("write header for file %s: %w", f.Name, err))
			continue
		}

		_, err = tarWriter.Write(f.Content)
		if err != nil {
			errs = append(errs, fmt.Errorf("write contents for file %s: %w", f.Name, err))
			continue
		}
	}
	defer func() {
		closeErr := tarWriter.Close()
		if closeErr != nil {
			err = fmt.Errorf("close tar writer: %w", closeErr)
		}
	}()

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return bytes.NewReader(buf.Bytes()), nil
}

// ReaderFromDir creates a TAR archive reader from a directory.
// It returns an error if the directory cannot be read or if the files cannot be read.
// This function is useful for creating a build context to build an image.
func ReaderFromDir(dir string) (r io.ReadSeeker, err error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	buildFiles := make([]BuildFile, 0, len(files))

	// walk the directory and add all files to the build context
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		contents, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, fmt.Errorf("read file %s: %w", f.Name(), err)
		}

		buildFiles = append(buildFiles, BuildFile{
			Name:    f.Name(),
			Content: contents,
		})
	}

	return ReaderFromFiles(buildFiles)
}

// ImageBuildClient is a client that can build images.
type ImageBuildClient interface {
	ImageClient

	// ImageBuild builds an image from a build context and options.
	ImageBuild(ctx context.Context, options build.ImageBuildOptions) (build.ImageBuildResponse, error)
}

// Build will build and image from context and Dockerfile, then return the tag
func Build(ctx context.Context, contextReader io.ReadSeeker, tag string, opts ...BuildOption) (string, error) {
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

	if contextReader == nil {
		return "", errors.New("context reader is required")
	}
	// Set the passed context reader, even if it is set in the build options.
	buildOpts.opts.Context = contextReader

	if buildOpts.buildClient == nil {
		buildOpts.buildClient = client.DefaultClient
		// In case there is no build client set, use the default docker client
		// to build the image. Needs to be closed when done.
		defer buildOpts.buildClient.Close()
	}

	if buildOpts.logWriter == nil {
		// If no log writer is set, use the default log writer
		// to build the image.
		buildOpts.logWriter = os.Stdout
	}

	if buildOpts.opts.Labels == nil {
		buildOpts.opts.Labels = make(map[string]string)
	}

	// Add client labels
	client.AddSDKLabels(buildOpts.opts.Labels)

	resp, err := backoff.RetryNotifyWithData(
		func() (build.ImageBuildResponse, error) {
			var err error
			defer tryClose(contextReader) // release resources in any case

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

	output := buildOpts.logWriter

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
