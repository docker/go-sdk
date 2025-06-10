package dockercontainer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-sdk/dockerclient"
)

// File represents a file that will be copied when container starts
type File struct {
	// Reader the reader to read the file from
	Reader io.Reader

	// ContainerPath the path to the file in the container.
	// Use the slash character that matches the path separator of the operating system
	// for the container.
	ContainerPath string

	// Mode the mode of the file
	Mode int64
}

// validate validates the [File]
func (f *File) validate() error {
	if f.Reader == nil {
		return errors.New("reader must be specified")
	}

	if f.ContainerPath == "" {
		return errors.New("container path must be specified")
	}

	return nil
}

// FileFromContainer implements io.ReadCloser and tar.Reader for a single file in a container.
type FileFromContainer struct {
	underlying *io.ReadCloser
	tarreader  *tar.Reader
}

// Read reads the file from the container.
func (fc *FileFromContainer) Read(b []byte) (int, error) {
	return (*fc.tarreader).Read(b)
}

// Close closes the file from the container.
func (fc *FileFromContainer) Close() error {
	return (*fc.underlying).Close()
}

// CopyFromContainer copies a file from the container to the local filesystem.
func (c *Container) CopyFromContainer(ctx context.Context, containerFilePath string) (io.ReadCloser, error) {
	r, _, err := c.dockerClient.CopyFromContainer(ctx, c.ID, containerFilePath)
	if err != nil {
		return nil, err
	}

	tarReader := tar.NewReader(r)

	// if we got here we have exactly one file in the TAR-stream
	// so we advance the index by one so the next call to Read will start reading it
	_, err = tarReader.Next()
	if err != nil {
		return nil, err
	}

	ret := &FileFromContainer{
		underlying: &r,
		tarreader:  tarReader,
	}

	return ret, nil
}

// CopyToContainer copies fileContent data to a file in container
func (c *Container) CopyToContainer(ctx context.Context, fileContent []byte, containerFilePath string, fileMode int64) error {
	contentFn := func(tw io.Writer) error {
		_, err := tw.Write(fileContent)
		return fmt.Errorf("write file content: %w", err)
	}

	buffer, err := tarFile(containerFilePath, contentFn, int64(len(fileContent)), fileMode)
	if err != nil {
		return fmt.Errorf("tar file: %w", err)
	}

	dockerClient, err := dockerclient.New(ctx)
	if err != nil {
		return fmt.Errorf("new docker client: %w", err)
	}

	err = dockerClient.CopyToContainer(ctx, c.ID, "/", buffer, container.CopyToContainerOptions{})
	if err != nil {
		return fmt.Errorf("copy to container: %w", err)
	}

	return nil
}

// isDir checks if a path is a directory
func isDir(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return false, err
	}

	if fileInfo.IsDir() {
		return true, nil
	}

	return false, nil
}

// tarDir compress a directory using tar + gzip algorithms
func tarDir(logger *slog.Logger, src string, fileMode int64) (*bytes.Buffer, error) {
	// always pass src as absolute path
	abs, err := filepath.Abs(src)
	if err != nil {
		return &bytes.Buffer{}, fmt.Errorf("error getting absolute path: %w", err)
	}
	src = abs

	buffer := &bytes.Buffer{}

	logger.Info("creating TAR file", "src", src)

	// tar > gzip > buffer
	zr := gzip.NewWriter(buffer)
	tw := tar.NewWriter(zr)

	_, baseDir := filepath.Split(src)
	// keep the path relative to the parent directory
	index := strings.LastIndex(src, baseDir)

	// walk through every file in the folder
	err = filepath.Walk(src, func(file string, fi os.FileInfo, errFn error) error {
		if errFn != nil {
			return fmt.Errorf("walk the file system: %w", errFn)
		}

		// if a symlink, skip file
		if fi.Mode().Type() == os.ModeSymlink {
			logger.Warn("skipping symlink", "file", file)
			return nil
		}

		// generate tar header
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return fmt.Errorf("file info header: %w", err)
		}

		// see https://pkg.go.dev/archive/tar#FileInfoHeader:
		// Since fs.FileInfo's Name method only returns the base name of the file it describes,
		// it may be necessary to modify Header.Name to provide the full path name of the file.
		header.Name = filepath.ToSlash(file[index:])
		header.Mode = fileMode

		// write header
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("write header: %w", err)
		}

		// if not a dir, write file content
		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return fmt.Errorf("open file: %w", err)
			}
			defer data.Close()
			if _, err := io.Copy(tw, data); err != nil {
				return fmt.Errorf("copy: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return buffer, err
	}

	// produce tar
	if err := tw.Close(); err != nil {
		return buffer, fmt.Errorf("close tar file: %w", err)
	}
	// produce gzip
	if err := zr.Close(); err != nil {
		return buffer, fmt.Errorf("close gzip file: %w", err)
	}

	return buffer, nil
}

// tarFile compress a single file using tar + gzip algorithms
func tarFile(basePath string, fileContent func(tw io.Writer) error, fileContentSize int64, fileMode int64) (*bytes.Buffer, error) {
	buffer := &bytes.Buffer{}

	zr := gzip.NewWriter(buffer)
	tw := tar.NewWriter(zr)

	hdr := &tar.Header{
		Name: basePath,
		Mode: fileMode,
		Size: fileContentSize,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return buffer, err
	}
	if err := fileContent(tw); err != nil {
		return buffer, err
	}

	// produce tar
	if err := tw.Close(); err != nil {
		return buffer, fmt.Errorf("close tar file: %w", err)
	}
	// produce gzip
	if err := zr.Close(); err != nil {
		return buffer, fmt.Errorf("close gzip file: %w", err)
	}

	return buffer, nil
}
