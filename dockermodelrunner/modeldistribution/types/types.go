package types

import (
	"encoding/json"
	"fmt"
	"time"
)

// Store interface for model storage operations
type Store interface {
	// Push a model to the store with given tags
	Push(modelPath string, tags []string) error

	// Pull a model by tag
	Pull(tag string, destPath string) error

	// List all models in the store
	List() ([]Model, error)

	// GetByTag Get model info by tag
	GetByTag(tag string) (*Model, error)

	// Delete a model by tag
	Delete(tag string) error

	// AddTags Add tags to an existing model
	AddTags(tag string, newTags []string) error

	// RemoveTags Remove tags from a model
	RemoveTags(tags []string) error

	// Version Get store version
	Version() string

	// Upgrade store to latest version
	Upgrade() error
}

// Model represents a model with its metadata and tags
type Model struct {
	// ID is the globally unique model identifier.
	ID string `json:"id"`
	// Tags are the list of tags associated with the model.
	Tags []string `json:"tags"`
	// Files are the GGUF files associated with the model.
	Files []string `json:"files"`
	// Created is the Unix epoch timestamp corresponding to the model creation.
	Created time.Time `json:"created"`
}

// modelAlias is an alias for Model to avoid recursion in JSON marshaling/unmarshaling.
// This is necessary because we want Model to contain a time.Time field which is not directly
// compatible with JSON serialization/deserialization.
type modelAlias Model

// modelResponseJSON is a struct used for JSON marshaling/unmarshaling of Model.
// It includes a Unix timestamp for the Created field to ensure compatibility with JSON.
type modelResponseJSON struct {
	modelAlias
	CreatedAt int64 `json:"created"`
}

// UnmarshalJSON implements json.Unmarshaler.
func (mr *Model) UnmarshalJSON(b []byte) error {
	var resp modelResponseJSON
	if err := json.Unmarshal(b, &resp); err != nil {
		return fmt.Errorf("unmarshal model response: %w", err)
	}
	*mr = Model(resp.modelAlias)
	mr.Created = time.Unix(resp.CreatedAt, 0)
	return nil
}

// MarshalJSON implements json.Marshaler.
func (mr Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(modelResponseJSON{
		modelAlias: modelAlias(mr),
		CreatedAt:  mr.Created.Unix(),
	})
}

// ModelIndex represents the index of all models in the store
type ModelIndex struct {
	Models []Model `json:"models"`
}

// StoreLayout represents the layout information of the store
type StoreLayout struct {
	Version string `json:"version"`
}

// ManifestReference represents a reference to a manifest in the store
type ManifestReference struct {
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
}

// StoreOptions represents options for creating a store
type StoreOptions struct {
	RootPath string
}
