package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"id": "model123",
		"object": "model",
		"created": 1682179200,
		"owned_by": "docker"
	}`

	var response OpenAIModel
	err := json.Unmarshal([]byte(jsonData), &response)
	require.NoError(t, err)
	require.Equal(t, OpenAIModel{
		ID:      "model123",
		Object:  "model",
		Created: time.Unix(1682179200, 0),
		OwnedBy: "docker",
	}, response)
}

func TestUnmarshalJSONError(t *testing.T) {
	// Invalid JSON with malformed created timestamp
	invalidJSON := `{
        "id": "model123",
        "object": "model",
        "created": "not-a-number",
        "owned_by": "docker"
    }`

	var response OpenAIModel
	err := json.Unmarshal([]byte(invalidJSON), &response)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unmarshal model response")
}

func TestMarshalJSON(t *testing.T) {
	response := OpenAIModel{
		ID:      "model123",
		Object:  "model",
		Created: time.Unix(1682179200, 0),
		OwnedBy: "docker",
	}

	expectedJSON := `{"id":"model123","object":"model","created":1682179200,"owned_by":"docker"}`

	jsonData, err := json.Marshal(response)
	require.NoError(t, err, "Failed to marshal JSON")
	require.JSONEq(t, expectedJSON, string(jsonData), "Unexpected JSON output")
}
