package desktop

import (
	"encoding/json"
	"fmt"
	"time"
)

// ProgressMessage represents a message sent during model pull operations
type ProgressMessage struct {
	Type    string `json:"type"`    // "progress", "success", or "error"
	Message string `json:"message"` // Human-readable message
}

// OpenAIChatMessage represents a message sent during OpenAI chat operations
type OpenAIChatMessage struct {
	// Role is the role of the message sender.
	Role string `json:"role"`

	// Content is the content of the message.
	Content string `json:"content"`
}

// OpenAIChatRequest represents a request to the OpenAI chat API.
type OpenAIChatRequest struct {
	// Model is the model to use for the chat.
	Model string `json:"model"`

	// Messages is the list of messages to send to the chat.
	Messages []OpenAIChatMessage `json:"messages"`

	// Stream is whether to stream the response.
	Stream bool `json:"stream"`
}

// OpenAIChatResponse represents a response from the OpenAI chat API.
type OpenAIChatResponse struct {
	// ID is the ID of the chat.
	ID string `json:"id"`

	// Object is the object type.
	Object string `json:"object"`

	// Created is the creation time of the chat.
	Created int64 `json:"created"`

	// Model is the model used for the chat.
	Model string `json:"model"`

	// Choices is the list of choices from the chat.
	Choices []struct {
		// Delta is the delta of the choice.
		Delta struct {
			// Content is the content of the choice.
			Content string `json:"content"`

			// Role is the role of the choice.
			Role string `json:"role,omitempty"`
		} `json:"delta"`

		// Index is the index of the choice.
		Index int `json:"index"`

		// FinishReason is the reason the chat finished.
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// OpenAIModel represents a model in the OpenAI API.
type OpenAIModel struct {
	// ID is the ID of the model.
	ID string `json:"id"`

	// Object is the object type.
	Object string `json:"object"`

	// Created is the creation time of the model.
	Created int64 `json:"created"`

	// OwnedBy is the owner of the model.
	OwnedBy string `json:"owned_by"`
}

// OpenAIModelList represents a list of models in the OpenAI API.
type OpenAIModelList struct {
	// Object is the object type.
	Object string `json:"object"`

	// Data is the list of models.
	Data []*OpenAIModel `json:"data"`
}

// Format represents the format of a model.
// TODO: To be replaced by the Model struct from pianta's common/pkg/inference/models/api.go.
// (https://github.com/docker/pinata/pull/33331)
type Format string

// Config represents the configuration of a model.
type Config struct {
	// Format is the format of the model.
	Format Format `json:"format,omitempty"`

	// Quantization is the quantization of the model.
	Quantization string `json:"quantization,omitempty"`

	// Parameters is the parameters of the model.
	Parameters string `json:"parameters,omitempty"`

	// Architecture is the architecture of the model.
	Architecture string `json:"architecture,omitempty"`

	// Size is the size of the model.
	Size string `json:"size,omitempty"`
}

// Model represents a model in the Docker Model Runner.
type Model struct {
	// ID is the globally unique model identifier.
	ID string `json:"id"`

	// Tags are the list of tags associated with the model.
	Tags []string `json:"tags"`

	// Created is the Unix epoch timestamp corresponding to the model creation.
	Created time.Time `json:"created"`

	// Config describes the model.
	Config Config `json:"config"`
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
