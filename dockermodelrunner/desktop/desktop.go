package desktop

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/docker/docker-sdk-go/dockermodelrunner/inference"
	"github.com/docker/docker-sdk-go/dockermodelrunner/models"
)

var (
	// ErrNotFound is returned when a model is not found.
	ErrNotFound = errors.New("model not found")

	// ErrServiceUnavailable is returned when the service is unavailable.
	ErrServiceUnavailable = errors.New("service unavailable")
)

// Client is a client for the Docker Model Runner API.
type Client struct {
	dockerClient DockerHTTPClient
}

// DockerHTTPClient is an interface that can be used to mock the Docker client.
//
//go:generate mockgen -source=desktop.go -destination=../mocks/mock_desktop.go -package=mocks DockerHTTPClient
type DockerHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// New creates a new Client.
func New(dockerClient DockerHTTPClient) *Client {
	return &Client{dockerClient}
}

// Status represents the status of the Docker Model Runner.
type Status struct {
	Running bool   `json:"running"`
	Status  []byte `json:"status"`
	Error   error  `json:"error"`
}

// Status returns the status of the Docker Model Runner.
func (c *Client) Status() Status {
	// TODO: Query "/".
	resp, err := c.doRequest(http.MethodGet, inference.ModelsPrefix, nil)
	if err != nil {
		err = c.handleQueryError(err, inference.ModelsPrefix)
		if errors.Is(err, ErrServiceUnavailable) {
			return Status{
				Running: false,
			}
		}
		return Status{
			Running: false,
			Error:   err,
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Status{
			Running: false,
			Error:   fmt.Errorf("unexpected status code: %d", resp.StatusCode),
		}
	}

	var status []byte
	statusResp, err := c.doRequest(http.MethodGet, inference.InferencePrefix+"/status", nil)
	if err != nil {
		status = []byte(fmt.Sprintf("error querying status: %v", err))
	} else {
		defer statusResp.Body.Close()
		statusBody, err := io.ReadAll(statusResp.Body)
		if err != nil {
			status = []byte(fmt.Sprintf("error reading status body: %v", err))
		} else {
			status = statusBody
		}
	}
	return Status{
		Running: true,
		Status:  status,
	}
}

// Pull pulls a model from the Docker Model Runner.
func (c *Client) Pull(model string, progress func(string)) (string, bool, error) {
	jsonData, err := json.Marshal(models.ModelCreateRequest{From: model})
	if err != nil {
		return "", false, fmt.Errorf("marshal request: %w", err)
	}

	createPath := inference.ModelsPrefix + "/create"
	resp, err := c.doRequest(
		http.MethodPost,
		createPath,
		bytes.NewReader(jsonData),
	)
	if err != nil {
		return "", false, c.handleQueryError(err, createPath)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", false, fmt.Errorf("pull %s status=%d body=%s", model, resp.StatusCode, body)
	}

	return scanProgress(resp, "pull", model, progress)
}

// scanProgress scans the progress of a model for a given action.
func scanProgress(resp *http.Response, action string, model string, progress func(string)) (string, bool, error) {
	progressShown := false

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		progressLine := scanner.Text()
		if progressLine == "" {
			continue
		}

		// Parse the progress message
		var progressMsg ProgressMessage
		if err := json.Unmarshal([]byte(html.UnescapeString(progressLine)), &progressMsg); err != nil {
			return "", progressShown, fmt.Errorf("unmarshal progress message: %w", err)
		}

		// Handle different message types
		switch progressMsg.Type {
		case "progress":
			progress(progressMsg.Message)
			progressShown = true
		case "error":
			return "", progressShown, fmt.Errorf("%s %s: %s", action, model, progressMsg.Message)
		case "success":
			return progressMsg.Message, progressShown, nil
		default:
			return "", progressShown, fmt.Errorf("unknown message type: %s", progressMsg.Type)
		}
	}

	return "", progressShown, fmt.Errorf("%s model %s: unexpected end of stream", action, model)
}

// Push pushes a model to the Docker Model Runner.
func (c *Client) Push(model string, progress func(string)) (string, bool, error) {
	pushPath := inference.ModelsPrefix + "/" + model + "/push"
	resp, err := c.doRequest(
		http.MethodPost,
		pushPath,
		nil, // Assuming no body is needed for the push request
	)
	if err != nil {
		return "", false, c.handleQueryError(err, pushPath)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", false, fmt.Errorf("push %s status=%d body=%s", model, resp.StatusCode, body)
	}

	return scanProgress(resp, "push", model, progress)
}

// List lists all models in the Docker Model Runner.
func (c *Client) List() ([]Model, error) {
	modelsRoute := inference.ModelsPrefix
	body, err := c.listRaw(modelsRoute, "")
	if err != nil {
		return nil, err
	}

	var modelsJSON []Model
	if err := json.Unmarshal(body, &modelsJSON); err != nil {
		return nil, fmt.Errorf("unmarshal response body: %w", err)
	}

	return modelsJSON, nil
}

// ListOpenAI lists all models in the Docker Model Runner using the OpenAI API.
func (c *Client) ListOpenAI() (*OpenAIModelList, error) {
	modelsRoute := inference.InferencePrefix + "/v1/models"
	rawResponse, err := c.listRaw(modelsRoute, "")
	if err != nil {
		return nil, err
	}

	var modelsJSON OpenAIModelList
	if err := json.Unmarshal(rawResponse, &modelsJSON); err != nil {
		return nil, fmt.Errorf("unmarshal response body: %w", err)
	}

	return &modelsJSON, nil
}

// Inspect inspects a model in the Docker Model Runner.
func (c *Client) Inspect(model string) (*Model, error) {
	if model != "" {
		if !strings.Contains(strings.Trim(model, "/"), "/") {
			// Do an extra API call to check if the model parameter isn't a model ID.
			modelId, err := c.fullModelID(model)
			if err != nil {
				return nil, fmt.Errorf("invalid model name: %s", model)
			}
			model = modelId
		}
	}

	rawResponse, err := c.listRaw(fmt.Sprintf("%s/%s", inference.ModelsPrefix, model), model)
	if err != nil {
		return nil, err
	}

	var modelInspect Model
	if err := json.Unmarshal(rawResponse, &modelInspect); err != nil {
		return nil, fmt.Errorf("unmarshal response body: %w", err)
	}

	return &modelInspect, nil
}

// InspectOpenAI inspects a model in the Docker Model Runner using the OpenAI API.
func (c *Client) InspectOpenAI(model string) (*OpenAIModel, error) {
	modelsRoute := inference.InferencePrefix + "/v1/models"
	if !strings.Contains(strings.Trim(model, "/"), "/") {
		// Do an extra API call to check if the model parameter isn't a model ID.
		var err error
		if model, err = c.fullModelID(model); err != nil {
			return nil, fmt.Errorf("invalid model name: %s", model)
		}
	}

	rawResponse, err := c.listRaw(fmt.Sprintf("%s/%s", modelsRoute, model), model)
	if err != nil {
		return nil, err
	}

	var modelInspect OpenAIModel
	if err := json.Unmarshal(rawResponse, &modelInspect); err != nil {
		return nil, fmt.Errorf("unmarshal response body: %w", err)
	}

	return &modelInspect, nil
}

// listRaw lists all models in the Docker Model Runner.
func (c *Client) listRaw(route string, model string) ([]byte, error) {
	resp, err := c.doRequest(http.MethodGet, route, nil)
	if err != nil {
		return nil, c.handleQueryError(err, route)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if model != "" && resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, model)
		}
		return nil, fmt.Errorf("list models: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	return body, nil
}

// fullModelID returns the full model ID for a given model ID.
func (c *Client) fullModelID(id string) (string, error) {
	bodyResponse, err := c.listRaw(inference.ModelsPrefix, "")
	if err != nil {
		return "", err
	}

	var modelsJSON []Model
	if err := json.Unmarshal(bodyResponse, &modelsJSON); err != nil {
		return "", fmt.Errorf("unmarshal response body: %w", err)
	}

	for _, m := range modelsJSON {
		if m.ID[7:19] == id || strings.TrimPrefix(m.ID, "sha256:") == id || m.ID == id {
			return m.ID, nil
		}
	}

	return "", fmt.Errorf("model with ID %s not found", id)
}

// Chat chats with a model in the Docker Model Runner.
func (c *Client) Chat(model, prompt string) error {
	if !strings.Contains(strings.Trim(model, "/"), "/") {
		// Do an extra API call to check if the model parameter isn't a model ID.
		if expanded, err := c.fullModelID(model); err == nil {
			model = expanded
		}
	}

	reqBody := OpenAIChatRequest{
		Model: model,
		Messages: []OpenAIChatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	chatCompletionsPath := inference.InferencePrefix + "/v1/chat/completions"
	resp, err := c.doRequest(
		http.MethodPost,
		chatCompletionsPath,
		bytes.NewReader(jsonData),
	)
	if err != nil {
		return c.handleQueryError(err, chatCompletionsPath)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chat with %s status=%d body=%s", model, resp.StatusCode, body)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			break
		}

		var streamResp OpenAIChatResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			return fmt.Errorf("unmarshal stream response: %w", err)
		}

		if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta.Content != "" {
			chunk := streamResp.Choices[0].Delta.Content
			fmt.Print(chunk)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read response stream: %w", err)
	}

	return nil
}

// Remove removes a model from the Docker Model Runner.
func (c *Client) Remove(models []string, force bool) (string, error) {
	modelRemoved := ""
	for _, model := range models {
		// Check if not a model ID passed as parameter.
		if !strings.Contains(model, "/") {
			if expanded, err := c.fullModelID(model); err == nil {
				model = expanded
			}
		}

		// Construct the URL with query parameters
		removePath := fmt.Sprintf("%s/%s?force=%s",
			inference.ModelsPrefix,
			model,
			strconv.FormatBool(force),
		)

		resp, err := c.doRequest(http.MethodDelete, removePath, nil)
		if err != nil {
			return modelRemoved, c.handleQueryError(err, removePath)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == http.StatusNotFound {
				return modelRemoved, fmt.Errorf("no such model: %s", model)
			}
			var bodyStr string
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				bodyStr = fmt.Sprintf("(failed to read response body: %v)", err)
			} else {
				bodyStr = string(body)
			}
			return modelRemoved, fmt.Errorf("removing %s failed with status %s: %s", model, resp.Status, bodyStr)
		}
		modelRemoved += fmt.Sprintf("Model %s removed successfully\n", model)
	}
	return modelRemoved, nil
}

// URL returns the URL for the Docker Model Runner.
func URL(path string) string {
	return "http://localhost" + inference.ExperimentalEndpointsPrefix + path
}

// doRequest is a helper function that performs HTTP requests and handles 503 responses
func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, URL(path), body)
	if err != nil {
		return nil, fmt.Errorf("new %s request: %w", method, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.dockerClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusServiceUnavailable {
		resp.Body.Close()
		return nil, ErrServiceUnavailable
	}

	return resp, nil
}

// handleQueryError is a helper function that handles query errors.
func (c *Client) handleQueryError(err error, path string) error {
	if errors.Is(err, ErrServiceUnavailable) {
		return ErrServiceUnavailable
	}
	return fmt.Errorf("query %s: %w", path, err)
}

// Tag tags a model in the Docker Model Runner.
func (c *Client) Tag(source, targetRepo, targetTag string) (string, error) {
	// Check if the source is a model ID, and expand it if necessary
	if !strings.Contains(strings.Trim(source, "/"), "/") {
		// Do an extra API call to check if the model parameter might be a model ID
		if expanded, err := c.fullModelID(source); err == nil {
			source = expanded
		}
	}

	// Construct the URL with query parameters
	tagPath := fmt.Sprintf("%s/%s/tag?repo=%s&tag=%s",
		inference.ModelsPrefix,
		source,
		targetRepo,
		targetTag,
	)

	resp, err := c.doRequest(http.MethodPost, tagPath, nil)
	if err != nil {
		return "", c.handleQueryError(err, tagPath)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("tag %s:%s status=%d body=%s", targetRepo, targetTag, resp.StatusCode, body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}

	return string(body), nil
}
