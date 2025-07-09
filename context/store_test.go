package context

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractDockerHost(t *testing.T) {
	t.Run("context-found-with-host", func(t *testing.T) {
		host := requireDockerHost(t, "test-context", Context{
			Name: "test-context",
			Endpoints: map[string]*endpoint{
				"docker": {Host: "tcp://1.2.3.4:2375"},
			},
		})
		require.Equal(t, "tcp://1.2.3.4:2375", host)
	})

	t.Run("context-found-without-host", func(t *testing.T) {
		requireDockerHostError(t, "test-context", Context{
			Name: "test-context",
			Endpoints: map[string]*endpoint{
				"docker": {},
			},
		}, ErrDockerHostNotSet)
	})

	t.Run("context-not-found", func(t *testing.T) {
		requireDockerHostError(t, "missing", Context{
			Name: "other-context",
			Endpoints: map[string]*endpoint{
				"docker": {Host: "tcp://1.2.3.4:2375"},
			},
		}, ErrDockerContextNotFound)
	})

	t.Run("nested-context-found", func(t *testing.T) {
		host := requireDockerHostInPath(t, "nested-context", "parent/nested-context", Context{
			Name: "nested-context",
			Endpoints: map[string]*endpoint{
				"docker": {Host: "tcp://1.2.3.4:2375"},
			},
		})
		require.Equal(t, "tcp://1.2.3.4:2375", host)
	})
}

func TestStore_Inspect(t *testing.T) {
	tmpDir := t.TempDir()
	setupTestContext(t, tmpDir, "test", Context{
		Name: "test",
		Metadata: &Metadata{
			Description: "test context",
		},
		Endpoints: map[string]*endpoint{
			"docker": {
				Host: "tcp://localhost:2375",
			},
		},
	})

	s := &store{root: tmpDir}

	t.Run("inspect/1", func(tt *testing.T) {
		ctx, err := s.inspect("test")
		require.NoError(tt, err)
		require.Equal(tt, "test", ctx.Name)
		require.Equal(tt, "test context", ctx.Metadata.Description)
		require.Equal(tt, "tcp://localhost:2375", ctx.Endpoints["docker"].Host)
		require.False(tt, ctx.Endpoints["docker"].SkipTLSVerify)
	})

	t.Run("inspect/not-found", func(tt *testing.T) {
		ctx, err := s.inspect("not-found")
		require.ErrorIs(tt, err, ErrDockerContextNotFound)
		require.Empty(tt, ctx)
	})

	t.Run("inspect/with-fields", func(tt *testing.T) {
		// Create a dockerContext and set additional fields using the new methods
		dockerCtx := &Metadata{
			Description: "ctx with fields",
		}
		dockerCtx.SetField("otel", map[string]any{
			"OTEL_EXPORTER_OTLP_ENDPOINT": "unix:///Users/mdelapenya/.docker/cloud/daemon.grpc.sock",
			"OTEL_EXPORTER_OTLP_PROTOCOL": "grpc",
		})

		setupTestContext(t, tmpDir, "fields", Context{
			Name:     "ctx-with-fields",
			Metadata: dockerCtx,
			Endpoints: map[string]*endpoint{
				"docker": {
					Host: "tcp://localhost:2375",
				},
			},
		})

		ctx, err := s.inspect("ctx-with-fields")
		require.NoError(tt, err)
		require.Equal(tt, "ctx-with-fields", ctx.Name)
		require.Equal(tt, "ctx with fields", ctx.Metadata.Description)
		require.Equal(tt, "tcp://localhost:2375", ctx.Endpoints["docker"].Host)
		require.False(tt, ctx.Endpoints["docker"].SkipTLSVerify)

		// Verify additional fields are accessible
		otelValue, exists := ctx.Metadata.Field("otel")
		require.True(tt, exists)
		require.NotNil(tt, otelValue)

		// Verify the structure of the otel field
		otelMap, ok := otelValue.(map[string]any)
		require.True(tt, ok)
		require.Len(tt, otelMap, 2)
		require.Equal(tt, "unix:///Users/mdelapenya/.docker/cloud/daemon.grpc.sock", otelMap["OTEL_EXPORTER_OTLP_ENDPOINT"])
		require.Equal(tt, "grpc", otelMap["OTEL_EXPORTER_OTLP_PROTOCOL"])
	})
}

func TestStore_List(t *testing.T) {
	t.Run("list/1", func(tt *testing.T) {
		tmpDir := t.TempDir()

		// Create a dockerContext and set additional fields using the new methods
		dockerCtx := &Metadata{
			Description: "test context",
		}
		dockerCtx.SetField("test", true)

		want := Context{
			Name:     "test",
			Metadata: dockerCtx,
			Endpoints: map[string]*endpoint{
				"docker": {
					Host:          "tcp://localhost:2375",
					SkipTLSVerify: true,
				},
			},
		}

		setupTestContext(tt, tmpDir, "test", want)

		s := &store{root: tmpDir}

		got, err := s.list()
		require.NoError(tt, err)
		require.Len(tt, got, 1)
		require.Equal(tt, "test", got[0].Name)
		require.Equal(tt, "test context", got[0].Metadata.Description)
		require.Equal(tt, "tcp://localhost:2375", got[0].Endpoints["docker"].Host)
		require.True(tt, got[0].Endpoints["docker"].SkipTLSVerify)

		// Verify additional fields
		wantTestField, _ := want.Metadata.Field("test")
		gotTestField, exists := got[0].Metadata.Field("test")
		require.True(tt, exists)
		require.Equal(tt, wantTestField, gotTestField)
	})

	t.Run("list/empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)
		t.Setenv("USERPROFILE", tmpDir) // Windows support

		tempMkdirAll(t, filepath.Join(tmpDir, ".docker"))

		s := &store{root: tmpDir}

		contexts, err := s.list()
		require.NoError(t, err)
		require.Empty(t, contexts)
	})
}

func TestStore_load(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		// Create a dockerContext and set additional fields using the new methods
		dockerCtx := &Metadata{
			Description: "test context",
		}
		dockerCtx.SetField("test", true)

		want := Context{
			Name:     "test",
			Metadata: dockerCtx,
			Endpoints: map[string]*endpoint{
				"docker": {
					Host:          "tcp://localhost:2375",
					SkipTLSVerify: true,
				},
			},
		}

		contextDir := filepath.Join(tmpDir, "test")
		setupTestContext(t, tmpDir, "test", want)

		got, err := s.load(contextDir)
		require.NoError(t, err)
		require.Equal(t, want.Name, got.Name)
		require.Equal(t, want.Metadata.Description, got.Metadata.Description)

		// Verify additional fields
		wantTestField, _ := want.Metadata.Field("test")
		gotTestField, exists := got.Metadata.Field("test")
		require.True(t, exists)
		require.Equal(t, wantTestField, gotTestField)

		require.Equal(t, want.Endpoints["docker"].Host, got.Endpoints["docker"].Host)
		require.Equal(t, want.Endpoints["docker"].SkipTLSVerify, got.Endpoints["docker"].SkipTLSVerify)
	})

	t.Run("directory-does-not-exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		nonExistentDir := filepath.Join(tmpDir, "does-not-exist")
		_, err := s.load(nonExistentDir)
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
	})

	t.Run("meta-json-does-not-exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		contextDir := filepath.Join(tmpDir, "empty")
		require.NoError(t, os.MkdirAll(contextDir, 0o755))

		_, err := s.load(contextDir)
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
	})

	t.Run("invalid-json", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		contextDir := filepath.Join(tmpDir, "invalid")
		require.NoError(t, os.MkdirAll(contextDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(contextDir, metaFile),
			[]byte("invalid json"),
			0o644,
		))

		_, err := s.load(contextDir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "parse metadata")
	})

	t.Run("permission-denied", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("permission tests not supported on Windows")
		}

		if os.Getuid() == 0 {
			t.Skip("cannot test permission denied as root")
		}

		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		contextDir := filepath.Join(tmpDir, "no-access")
		require.NoError(t, os.MkdirAll(contextDir, 0o755))

		meta := Context{
			Name: "test",
			Endpoints: map[string]*endpoint{
				"docker": {Host: "tcp://localhost:2375"},
			},
		}
		setupTestContext(t, tmpDir, "no-access", meta)

		// Remove read permissions
		require.NoError(t, os.Chmod(filepath.Join(contextDir, metaFile), 0o000))

		_, err := s.load(contextDir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "permission denied")
	})

	t.Run("windows-file-access-error", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("Windows-specific test")
		}

		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		contextDir := filepath.Join(tmpDir, "locked")
		require.NoError(t, os.MkdirAll(contextDir, 0o755))

		// Create and lock the file
		f, err := os.Create(filepath.Join(contextDir, metaFile))
		require.NoError(t, err)
		require.NoError(t, f.Close())

		// Try to load while file is locked
		f2, err := os.OpenFile(filepath.Join(contextDir, metaFile), os.O_RDWR, 0o644)
		require.NoError(t, err)
		defer f2.Close()

		_, err = s.load(contextDir)
		require.Error(t, err)
	})

	t.Run("empty-but-valid-json", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		contextDir := filepath.Join(tmpDir, "empty")
		require.NoError(t, os.MkdirAll(contextDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(contextDir, metaFile),
			[]byte("{}"),
			0o644,
		))

		got, err := s.load(contextDir)
		require.NoError(t, err)
		require.Empty(t, got.Name)
		require.Nil(t, got.Metadata)
		require.Empty(t, got.Endpoints)
	})

	t.Run("partial-metadata", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		contextDir := filepath.Join(tmpDir, "partial")
		require.NoError(t, os.MkdirAll(contextDir, 0o755))

		// Only name and docker endpoint, no context metadata
		meta := Context{
			Name: "test",
			Endpoints: map[string]*endpoint{
				"docker": {Host: "tcp://localhost:2375"},
			},
		}
		setupTestContext(t, tmpDir, "partial", meta)

		got, err := s.load(contextDir)
		require.NoError(t, err)
		require.Equal(t, "test", got.Name)
		require.Nil(t, got.Metadata)
		require.Equal(t, "tcp://localhost:2375", got.Endpoints["docker"].Host)
	})
}

func TestStore_list(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		// Setup test contexts
		contexts := map[string]Context{
			"context1": {
				Name: "context1",
				Endpoints: map[string]*endpoint{
					"docker": {Host: "tcp://1.2.3.4:2375"},
				},
			},
			"nested/context2": {
				Name: "context2",
				Endpoints: map[string]*endpoint{
					"docker": {Host: "unix:///var/run/docker.sock"},
				},
			},
		}

		for path, meta := range contexts {
			setupTestContext(t, tmpDir, path, meta)
		}

		list, err := s.list()
		require.NoError(t, err)
		require.Len(t, list, 2)
	})

	t.Run("root-does-not-exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonExistentDir := filepath.Join(tmpDir, "does-not-exist")
		s := &store{root: nonExistentDir}

		list, err := s.list()
		require.ErrorIs(t, err, os.ErrNotExist)
		require.Empty(t, list)
	})

	t.Run("corrupted-metadata-file", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		// Create a context directory with invalid JSON
		contextDir := filepath.Join(tmpDir, "invalid")
		require.NoError(t, os.MkdirAll(contextDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(contextDir, metaFile),
			[]byte("invalid json"),
			0o644,
		))

		_, err := s.list()
		require.Error(t, err)
		require.Contains(t, err.Error(), "parse metadata")
	})

	t.Run("mixed-valid-and-invalid-contexts", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		// Setup one valid context
		validMeta := Context{
			Name: "valid",
			Endpoints: map[string]*endpoint{
				"docker": {Host: "tcp://1.2.3.4:2375"},
			},
		}
		setupTestContext(t, tmpDir, "valid", validMeta)

		// Setup an invalid context
		invalidDir := filepath.Join(tmpDir, "invalid")
		require.NoError(t, os.MkdirAll(invalidDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(invalidDir, metaFile),
			[]byte("invalid json"),
			0o644,
		))

		_, err := s.list()
		require.Error(t, err)
		require.Contains(t, err.Error(), "parse metadata")
	})

	t.Run("permission-denied", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("permission tests not supported on Windows")
			return
		}

		if os.Getuid() == 0 {
			t.Skip("cannot test permission denied as root")
		}

		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		// Create a context with no read permissions
		contextDir := filepath.Join(tmpDir, "no-access")
		require.NoError(t, os.MkdirAll(contextDir, 0o755))

		meta := Context{
			Name: "test",
			Endpoints: map[string]*endpoint{
				"docker": {Host: "tcp://1.2.3.4:2375"},
			},
		}
		setupTestContext(t, tmpDir, "no-access", meta)

		// Remove read permissions
		require.NoError(t, os.Chmod(filepath.Join(contextDir, metaFile), 0o000))

		list, err := s.list()
		require.Error(t, err)
		require.Contains(t, err.Error(), "permission denied")
		require.Empty(t, list)
	})

	t.Run("windows-file-access-error", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("Windows-specific test")
			return
		}

		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		contextDir := filepath.Join(tmpDir, "locked")
		require.NoError(t, os.MkdirAll(contextDir, 0o755))

		// Create and lock the file
		f, err := os.Create(filepath.Join(contextDir, metaFile))
		require.NoError(t, err)
		require.NoError(t, f.Close())

		// Try to list while file is locked
		f2, err := os.OpenFile(filepath.Join(contextDir, metaFile), os.O_RDWR, 0o644)
		require.NoError(t, err)
		defer f2.Close()

		list, err := s.list()
		require.Error(t, err)
		require.Empty(t, list)
	})

	t.Run("empty-but-valid-context-file", func(t *testing.T) {
		tmpDir := t.TempDir()
		s := &store{root: tmpDir}

		// Create a context with empty but valid JSON
		contextDir := filepath.Join(tmpDir, "empty")
		require.NoError(t, os.MkdirAll(contextDir, 0o755))
		require.NoError(t, os.WriteFile(
			filepath.Join(contextDir, metaFile),
			[]byte("{}"),
			0o644,
		))

		list, err := s.list()
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.Empty(t, list[0].Name)
		require.Empty(t, list[0].Endpoints)
	})
}

func TestDockerContext_JSON_Marshaling(t *testing.T) {
	t.Run("marshal-with-additional-fields", func(t *testing.T) {
		// Create a dockerContext with additional fields
		dockerCtx := &Metadata{
			Description: "test context with fields",
		}
		dockerCtx.SetField("otel", map[string]any{
			"OTEL_EXPORTER_OTLP_ENDPOINT": "unix:///socket.sock",
			"OTEL_EXPORTER_OTLP_PROTOCOL": "grpc",
		})
		dockerCtx.SetField("cloud.docker.com", map[string]any{
			"accountName": "test-account",
		})

		// Marshal to JSON
		data, err := json.Marshal(dockerCtx)
		require.NoError(t, err)

		// Verify the JSON structure - additional fields should be at the same level as Description
		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		// Description should be at the top level
		require.Equal(t, "test context with fields", result["Description"])

		// Additional fields should be at the same level, not nested under "Fields"
		require.Contains(t, result, "otel")
		require.Contains(t, result, "cloud.docker.com")
		require.NotContains(t, result, "Fields") // Should NOT have a Fields key

		// Verify otel structure
		otelValue, ok := result["otel"].(map[string]any)
		require.True(t, ok)
		require.Len(t, otelValue, 2)
		require.Equal(t, "unix:///socket.sock", otelValue["OTEL_EXPORTER_OTLP_ENDPOINT"])
		require.Equal(t, "grpc", otelValue["OTEL_EXPORTER_OTLP_PROTOCOL"])

		// Verify cloud.docker.com structure
		cloudValue, ok := result["cloud.docker.com"].(map[string]any)
		require.True(t, ok)
		require.Len(t, cloudValue, 1)
		require.Equal(t, "test-account", cloudValue["accountName"])
	})

	t.Run("unmarshal-with-additional-fields", func(t *testing.T) {
		// JSON data with additional fields at the same level as Description
		jsonData := `{
			"Description": "test context",
			"otel": {
				"OTEL_EXPORTER_OTLP_ENDPOINT": "unix:///socket.sock",
				"OTEL_EXPORTER_OTLP_PROTOCOL": "grpc"
			},
			"cloud.docker.com": {
				"accountName": "test-account"
			}
		}`

		var dockerCtx Metadata
		err := json.Unmarshal([]byte(jsonData), &dockerCtx)
		require.NoError(t, err)

		// Verify Description
		require.Equal(t, "test context", dockerCtx.Description)

		fields := dockerCtx.Fields()
		require.Len(t, fields, 2)
		require.Contains(t, fields, "otel")
		require.Contains(t, fields, "cloud.docker.com")

		// description is not a field, it's a top-level field
		description, exists := dockerCtx.Field("Description")
		require.False(t, exists)
		require.Empty(t, description)

		// Verify additional fields
		otelValue, exists := dockerCtx.Field("otel")
		require.True(t, exists)
		require.Len(t, otelValue, 2)
		otelMap, ok := otelValue.(map[string]any)
		require.True(t, ok)
		require.Equal(t, "unix:///socket.sock", otelMap["OTEL_EXPORTER_OTLP_ENDPOINT"])
		require.Equal(t, "grpc", otelMap["OTEL_EXPORTER_OTLP_PROTOCOL"])

		cloudValue, exists := dockerCtx.Field("cloud.docker.com")
		require.True(t, exists)
		require.Len(t, cloudValue, 1)
		cloudMap, ok := cloudValue.(map[string]any)
		require.True(t, ok)
		require.Equal(t, "test-account", cloudMap["accountName"])
	})

	t.Run("marshal-unmarshal-roundtrip", func(t *testing.T) {
		// Create original dockerContext
		original := &Metadata{
			Description: "roundtrip test",
		}
		original.SetField("custom", "value")
		original.SetField("complex", map[string]any{
			"nested": true,
			"count":  42,
		})

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)

		// Unmarshal back
		var restored Metadata
		err = json.Unmarshal(data, &restored)
		require.NoError(t, err)

		// Verify everything matches
		require.Equal(t, original.Description, restored.Description)

		customValue, exists := restored.Field("custom")
		require.True(t, exists)
		require.Equal(t, "value", customValue)

		complexValue, exists := restored.Field("complex")
		require.True(t, exists)
		require.Len(t, complexValue, 2)
		complexMap, ok := complexValue.(map[string]any)
		require.True(t, ok)
		require.Equal(t, true, complexMap["nested"])
		require.Equal(t, float64(42), complexMap["count"]) // JSON numbers are float64
	})
}

// requireDockerHost creates a context and verifies host extraction succeeds
func requireDockerHost(t *testing.T, contextName string, ctx Context) string {
	t.Helper()
	tmpDir := t.TempDir()

	setupTestContext(t, tmpDir, contextName, ctx)

	s := &store{root: tmpDir}

	ctx, err := s.inspect(contextName)
	require.NoError(t, err)
	return ctx.Endpoints["docker"].Host
}

// requireDockerHostInPath creates a context at a specific path and verifies host extraction
func requireDockerHostInPath(t *testing.T, contextName, path string, ctx Context) string {
	t.Helper()
	tmpDir := t.TempDir()

	setupTestContext(t, tmpDir, path, ctx)

	s := &store{root: tmpDir}

	ctx, err := s.inspect(contextName)
	require.NoError(t, err)
	return ctx.Endpoints["docker"].Host
}

// requireDockerHostError creates a context and verifies expected error
func requireDockerHostError(t *testing.T, contextName string, ctx Context, wantErr error) {
	t.Helper()
	tmpDir := t.TempDir()

	setupTestContext(t, tmpDir, contextName, ctx)

	s := &store{root: tmpDir}

	_, err := s.inspect(contextName)
	require.ErrorIs(t, err, wantErr)
}

// setupTestContext creates a test context file in the specified location
func setupTestContext(tb testing.TB, root, relPath string, ctx Context) {
	tb.Helper()

	contextDir := filepath.Join(root, relPath)
	require.NoError(tb, os.MkdirAll(contextDir, 0o755))

	data, err := json.Marshal(ctx)
	require.NoError(tb, err)

	require.NoError(tb, os.WriteFile(
		filepath.Join(contextDir, metaFile),
		data,
		0o644,
	))
}
