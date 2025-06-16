package container

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFile_validate(t *testing.T) {
	t.Run("empty-reader-and-host-path", func(t *testing.T) {
		file := File{}
		err := file.validate()
		require.Error(t, err)
	})

	t.Run("empty-reader", func(t *testing.T) {
		file := File{
			HostPath:      "test",
			ContainerPath: "/test",
		}
		err := file.validate()
		require.NoError(t, err)
	})

	t.Run("empty-container-path", func(t *testing.T) {
		file := File{
			Reader: bytes.NewReader([]byte("test")),
		}
		err := file.validate()
		require.Error(t, err)
	})

	t.Run("valid", func(t *testing.T) {
		file := File{
			Reader:        bytes.NewReader([]byte("test")),
			ContainerPath: "test",
		}
		err := file.validate()
		require.NoError(t, err)
	})
}

func TestIsDir(t *testing.T) {
	testIsDir := func(t *testing.T, filePath string, expected bool, expectedErr error) {
		t.Helper()

		result, err := isDir(filePath)
		if expectedErr != nil {
			require.Error(t, err, "expected error")
		} else {
			require.NoError(t, err, "not expected error")
		}
		require.Equal(t, expected, result)
	}

	t.Run("dir", func(t *testing.T) {
		testIsDir(t, "testdata", true, nil)
	})

	t.Run("dile", func(t *testing.T) {
		testIsDir(t, path.Join("testdata/Dockerfile"), false, nil)
	})

	t.Run("not-exist", func(t *testing.T) {
		testIsDir(t, "does-not-exist.go", false, errors.New("does not exist"))
	})
}

func TestTarDir(t *testing.T) {
	originalSrc := filepath.Join(".", "testdata")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testTarDir := func(t *testing.T, abs bool) {
		t.Helper()

		src := originalSrc
		if abs {
			absSrc, err := filepath.Abs(src)
			require.NoError(t, err)

			src = absSrc
		}

		buff, err := tarDir(logger, src, 0o755)
		require.NoError(t, err)

		tmpDir := filepath.Join(t.TempDir(), "subfolder")
		err = untar(tmpDir, bytes.NewReader(buff.Bytes()))
		require.NoError(t, err)

		srcFiles, err := os.ReadDir(src)
		require.NoError(t, err)

		for _, srcFile := range srcFiles {
			if srcFile.IsDir() {
				continue
			}
			srcBytes, err := os.ReadFile(filepath.Join(src, srcFile.Name()))
			require.NoError(t, err)

			untarBytes, err := os.ReadFile(filepath.Join(tmpDir, "testdata", srcFile.Name()))
			require.NoError(t, err)
			require.Equal(t, srcBytes, untarBytes)
		}
	}

	t.Run("absolute", func(t *testing.T) {
		testTarDir(t, true)
	})
	t.Run("relative", func(t *testing.T) {
		testTarDir(t, false)
	})
}

func TestTarFile(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(".", "testdata", "Dockerfile"))
	require.NoError(t, err)

	buff, err := tarFile("Docker.file", func(tw io.Writer) error {
		_, err := tw.Write(b)
		return err
	}, int64(len(b)), 0o755)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	err = untar(tmpDir, bytes.NewReader(buff.Bytes()))
	require.NoError(t, err)

	untarBytes, err := os.ReadFile(filepath.Join(tmpDir, "Docker.file"))
	require.NoError(t, err)
	require.Equal(t, b, untarBytes)
}

// untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func untar(dst string, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if it's a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0o755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; deferring would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
}
