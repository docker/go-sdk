package context

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
)

// Context represents a Docker context
type Context struct {
	metadata
}

// store manages Docker context metadata files
type store struct {
	root string
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

// dockerContext represents the metadata stored for a context
type dockerContext struct {
	// Description is the description of the context
	Description string `json:"Description,omitempty"`

	// additionalFields holds any additional fields that are not part of the standard metadata.
	// These are marshaled/unmarshaled at the same level as Description, not nested under a "Fields" key.
	additionalFields map[string]any
}

// MarshalJSON implements custom JSON marshaling for dockerContext
func (dc *dockerContext) MarshalJSON() ([]byte, error) {
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
func (dc *dockerContext) UnmarshalJSON(data []byte) error {
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
func (dc *dockerContext) Field(key string) (any, bool) {
	if dc.additionalFields == nil {
		return nil, false
	}
	value, exists := dc.additionalFields[key]
	return value, exists
}

// SetField sets the value of an additional field
func (dc *dockerContext) SetField(key string, value any) {
	if dc.additionalFields == nil {
		dc.additionalFields = make(map[string]any)
	}
	dc.additionalFields[key] = value
}

// Fields returns a copy of all additional fields
func (dc *dockerContext) Fields() map[string]any {
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

			return Context{
				metadata: *ctx,
			}, nil
		}
	}

	return Context{}, ErrDockerContextNotFound
}

func (s *store) list() ([]*metadata, error) {
	dirs, err := s.findMetadataDirs(s.root)
	if err != nil {
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
