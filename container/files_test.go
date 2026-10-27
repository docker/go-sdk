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

func TestUntar_rejectsPathTraversal(t *testing.T) {
	tests := []struct {
		name     string
		typeflag byte
	}{
		{name: "../evil.txt", typeflag: tar.TypeReg},
		{name: "nested/../../evil.txt", typeflag: tar.TypeReg},
		{name: "/evil.txt", typeflag: tar.TypeReg},
		{name: "../evil-dir", typeflag: tar.TypeDir},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			content := []byte("evil")

			var buf bytes.Buffer
			gw := gzip.NewWriter(&buf)
			tw := tar.NewWriter(gw)
			hdr := &tar.Header{Name: tc.name, Typeflag: tc.typeflag, Mode: 0o644}
			if tc.typeflag == tar.TypeReg {
				hdr.Size = int64(len(content))
			}
			require.NoError(t, tw.WriteHeader(hdr))
			if tc.typeflag == tar.TypeReg {
				_, err := tw.Write(content)
				require.NoError(t, err)
			}
			require.NoError(t, tw.Close())
			require.NoError(t, gw.Close())

			parent := t.TempDir()
			dst := filepath.Join(parent, "dst")
			require.NoError(t, os.Mkdir(dst, 0o755))

			err := untar(dst, &buf)
			require.Error(t, err)

			_, err = os.Stat(filepath.Join(parent, filepath.Base(tc.name)))
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	}
}

func TestUntar_specialModeBits(t *testing.T) {
	b := []byte("content")

	// os.Root rejects permission bits outside 0o777, so setuid/setgid/sticky
	// bits must be masked before extracting.
	buff, err := tarFile("special.file", func(tw io.Writer) error {
		_, err := tw.Write(b)
		return err
	}, int64(len(b)), 0o4755)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	require.NoError(t, untar(tmpDir, bytes.NewReader(buff.Bytes())))

	untarBytes, err := os.ReadFile(filepath.Join(tmpDir, "special.file"))
	require.NoError(t, err)
	require.Equal(t, b, untarBytes)
}

// untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files.
// All filesystem operations go through an os.Root scoped to 'dst', so entries
// whose names escape it (e.g. "../evil" or absolute paths) are rejected.
func untar(dst string, r io.Reader) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	root, err := os.OpenRoot(dst)
	if err != nil {
		return err
	}
	defer root.Close()

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

		// check the file type
		switch header.Typeflag {

		// if it's a dir, create it (and any missing parents) inside the root
		case tar.TypeDir:
			if err := root.MkdirAll(header.Name, 0o755); err != nil {
				return err
			}

		// if it's a file create it inside the root
		case tar.TypeReg:
			f, err := root.OpenFile(header.Name, os.O_CREATE|os.O_RDWR, header.FileInfo().Mode().Perm())
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}

			// manually close here after each file operation; deferring would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
}
