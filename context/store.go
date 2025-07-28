package context

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"github.com/opencontainers/go-digest"

	"github.com/docker/go-sdk/config"
)

// Context represents a Docker context
type Context struct {
	// Name is the name of the context
	Name string `json:"Name,omitempty"`

	// encodedName is the digest of the context name
	encodedName string `json:"-"`

	// isCurrent is true if the context is the current context
	isCurrent bool `json:"-"`

	// Metadata is the metadata stored for a context
	Metadata *Metadata `json:"Metadata,omitempty"`

	// Endpoints is the list of endpoints for the context
	Endpoints map[string]*endpoint `json:"Endpoints,omitempty"`
}

// store manages Docker context metadata files
type store struct {
	root string
}

// Metadata represents the metadata stored for a context
type Metadata struct {
	// Description is the description of the context
	Description string `json:"Description,omitempty"`

	// additionalFields holds any additional fields that are not part of the standard metadata.
	// These are marshaled/unmarshaled at the same level as Description, not nested under a "Fields" key.
	additionalFields map[string]any
}

// MarshalJSON implements custom JSON marshaling for dockerContext
func (dc *Metadata) MarshalJSON() ([]byte, error) {
	// Pre-allocate with capacity for additional fields + Description field
	result := make(map[string]any, len(dc.additionalFields)+1)

	// Add Description if not empty
	if dc.Description != "" {
		result["Description"] = dc.Description
	}

	// Add all additional fields at the same level
	for key, value := range dc.additionalFields {
		result[key] = value
	}

	return json.Marshal(result)
}

// UnmarshalJSON implements custom JSON unmarshaling for dockerContext
func (dc *Metadata) UnmarshalJSON(data []byte) error {
	// First unmarshal into a generic map
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Extract known fields
	if desc, ok := raw["Description"]; ok {
		if descStr, ok := desc.(string); ok {
			dc.Description = descStr
		}
		delete(raw, "Description")
	}

	// Store remaining fields as additional fields
	dc.additionalFields = raw

	return nil
}

// Field returns the value of an additional field
func (dc *Metadata) Field(key string) (any, bool) {
	if dc.additionalFields == nil {
		return nil, false
	}
	value, exists := dc.additionalFields[key]
	return value, exists
}

// SetField sets the value of an additional field
func (dc *Metadata) SetField(key string, value any) {
	if dc.additionalFields == nil {
		dc.additionalFields = make(map[string]any)
	}
	dc.additionalFields[key] = value
}

// Fields returns a copy of all additional fields
func (dc *Metadata) Fields() map[string]any {
	if dc.additionalFields == nil {
		return make(map[string]any)
	}
	// Return a copy to prevent external modification
	result := make(map[string]any, len(dc.additionalFields))

	maps.Copy(result, dc.additionalFields)

	return result
}

// endpoint represents a Docker endpoint configuration
type endpoint struct {
	// Host is the host of the endpoint
	Host string `json:",omitempty"`

	// SkipTLSVerify is the flag to skip TLS verification
	SkipTLSVerify bool
}

// inspect inspects a context by name
func (s *store) inspect(ctxName string) (Context, error) {
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

			cfg, err := config.Load()
			if err != nil {
				return Context{}, fmt.Errorf("load config: %w", err)
			}
			ctx.isCurrent = cfg.CurrentContext == ctx.Name

			ctx.encodedName = digest.FromString(ctx.Name).Encoded()

			return *ctx, nil
		}
	}

	return Context{}, ErrDockerContextNotFound
}

// add adds a context to the store, creating the directory if it doesn't exist.
func (s *store) add(ctx *Context) error {
	if ctx.encodedName == "" {
		// it's fine to calculate the encoded name here because the context is not yet added to the store
		ctx.encodedName = digest.FromString(ctx.Name).Encoded()
	}

	if fileExists(filepath.Join(s.root, ctx.encodedName)) {
		return fmt.Errorf("context already exists: %s", ctx.Name)
	}

	err := os.MkdirAll(filepath.Join(s.root, ctx.encodedName), 0o755)
	if err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	data, err := json.Marshal(ctx)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	if err := os.WriteFile(filepath.Join(s.root, ctx.encodedName, metaFile), data, 0o644); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

// delete deletes a context from the store.
// The encoded name is the digest of the context name.
func (s *store) delete(encodedName string) error {
	return os.RemoveAll(filepath.Join(s.root, encodedName))
}

// list lists all contexts in the store
func (s *store) list() ([]*Context, error) {
	dirs, err := s.findMetadataDirs(s.root)
	if err != nil {
		return nil, fmt.Errorf("find contexts: %w", err)
	}

	var contexts []*Context
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

// load loads a context from a directory
func (s *store) load(dir string) (*Context, error) {
	data, err := os.ReadFile(filepath.Join(dir, metaFile))
	if err != nil {
		return nil, err
	}

	var meta Context
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parse metadata: %w", err)
	}
	return &meta, nil
}

// findMetadataDirs finds all metadata directories in the store,
// checking for the presence of a meta.json file in each directory.
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

// hasMetaFile checks if a directory contains a meta.json file
func hasMetaFile(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, metaFile))
	return err == nil && !info.IsDir()
}
