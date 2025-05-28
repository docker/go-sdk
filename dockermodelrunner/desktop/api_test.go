package desktop

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"id": "model123",
		"tags": ["tag1", "tag2"],
		"config": {
			"format": "format1",
			"quantization": "quantization1",
			"parameters": "parameters1",
			"architecture": "architecture1",
			"size": "size1"
		},
		"created": 1682179200
	}`

	var response Model
	err := json.Unmarshal([]byte(jsonData), &response)
	require.NoError(t, err)
	require.Equal(t, Model{
		ID:   "model123",
		Tags: []string{"tag1", "tag2"},
		Config: Config{
			Format:       "format1",
			Quantization: "quantization1",
			Parameters:   "parameters1",
			Architecture: "architecture1",
			Size:         "size1",
		},
		Created: time.Unix(1682179200, 0),
	}, response)
}

func TestUnmarshalJSONError(t *testing.T) {
	// Invalid JSON with malformed created timestamp
	invalidJSON := `{
        "id": "model123",
        "tags": ["tag1", "tag2"],
        "config": {
			"format": "format1",
			"quantization": "quantization1",
			"parameters": "parameters1",
			"architecture": "architecture1",
			"size": "size1"
		},
        "created": "not-a-number"
    }`

	var response Model
	err := json.Unmarshal([]byte(invalidJSON), &response)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unmarshal model response")
}

func TestMarshalJSON(t *testing.T) {
	response := Model{
		ID:   "model123",
		Tags: []string{"tag1", "tag2"},
		Config: Config{
			Format:       "format1",
			Quantization: "quantization1",
			Parameters:   "parameters1",
			Architecture: "architecture1",
			Size:         "size1",
		},
		Created: time.Unix(1682179200, 0),
	}

	expectedJSON := `{"id":"model123","tags":["tag1","tag2"],"config":{"format":"format1","quantization":"quantization1","parameters":"parameters1","architecture":"architecture1","size":"size1"},"created":1682179200}`

	jsonData, err := json.Marshal(response)
	require.NoError(t, err, "Failed to marshal JSON")
	require.JSONEq(t, expectedJSON, string(jsonData), "Unexpected JSON output")
}
