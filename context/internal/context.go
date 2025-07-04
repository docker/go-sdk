package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const metaFile = "meta.json"

// Context represents a Docker context
type Context struct {
	metadata
}

// dockerContext represents the metadata stored for a context
type dockerContext struct {
	// Description is the description of the context
	Description string

	// Fields is the additional fields of the context, holding any additional
	// fields that are not part of the standard metadata.
	Fields map[string]any
}

// endpoint represents a Docker endpoint configuration
type endpoint struct {
	// Host is the host of the endpoint
	Host string `json:",omitempty"`

	// SkipTLSVerify is the flag to skip TLS verification
	SkipTLSVerify bool
}

// metadata represents a complete context configuration
type metadata struct {
	// Name is the name of the context
	Name string `json:",omitempty"`

	// Context is the metadata stored for a context
	Context *dockerContext `json:"metadata,omitempty"`

	// Endpoints is the list of endpoints for the context
	Endpoints map[string]*endpoint `json:"endpoints,omitempty"`
}

// store manages Docker context metadata files
type store struct {
	root string
}

// Inspect returns the description of the given context.
// It returns an error if the context is not found or if the docker endpoint is not set.
func Inspect(ctxName string, metaRoot string) (Context, error) {
	s := &store{root: metaRoot}

	contexts, err := s.list()
	if err != nil {
		return Context{}, fmt.Errorf("list contexts: %w", err)
	}

	for _, ctx := range contexts {
		if ctx.Name == ctxName {
			ep, ok := ctx.Endpoints["docker"]
			if !ok || ep == nil || ep.Host == "" {
				return Context{}, ErrDockerHostNotSet
			}

			return Context{
				metadata: *ctx,
			}, nil
		}
	}

	return Context{}, ErrDockerContextNotFound
}

// List returns the list of contexts available in the Docker configuration.
func List(metaRoot string) ([]string, error) {
	s := &store{root: metaRoot}

	contexts, err := s.list()
	if err != nil {
		return nil, fmt.Errorf("list contexts: %w", err)
	}

	names := make([]string, len(contexts))
	for i, ctx := range contexts {
		names[i] = ctx.Name
	}
	return names, nil
}

func (s *store) list() ([]*metadata, error) {
	dirs, err := s.findMetadataDirs(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("find contexts: %w", err)
	}

	var contexts []*metadata
	for _, dir := range dirs {
		ctx, err := s.load(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("load context %s: %w", dir, err)
		}
		contexts = append(contexts, ctx)
	}
	return contexts, nil
}

func (s *store) load(dir string) (*metadata, error) {
	data, err := os.ReadFile(filepath.Join(dir, metaFile))
	if err != nil {
		return nil, err
	}

	var meta metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parse metadata: %w", err)
	}
	return &meta, nil
}

func (s *store) findMetadataDirs(root string) ([]string, error) {
	var dirs []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if hasMetaFile(path) {
				dirs = append(dirs, path)
				return filepath.SkipDir // don't recurse into context dirs
			}
		}
		return nil
	})
	return dirs, err
}

func hasMetaFile(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, metaFile))
	return err == nil && !info.IsDir()
}
