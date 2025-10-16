package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/docker/docker/api/types/registry"
)

// Errors from credential helpers.
var (
	ErrCredentialsNotFound         = errors.New("credentials not found in native keychain")
	ErrCredentialsMissingServerURL = errors.New("no credentials server URL")
)

//nolint:gochecknoglobals // These are used to mock exec in tests.
var (
	// execLookPath is a variable that can be used to mock exec.LookPath in tests.
	execLookPath = exec.LookPath
	// execCommand is a variable that can be used to mock exec.Command in tests.
	execCommand = exec.Command
)

// credentialsFromHelper attempts to lookup credentials from the passed in docker credential helper.
//
// The credential helper should just be the suffix name (no "docker-credential-").
// If the passed in helper program is empty this will look up the default helper for the platform.
//
// If the credentials are not found, no error is returned, only empty credentials.
//
// Hostnames should already be resolved using [ResolveRegistryHost]
//
// If the username string is empty, the password string is an identity token.
func credentialsFromHelper(helper, hostname string) (registry.AuthConfig, error) {
	var creds registry.AuthConfig
	credHelperName := helper
	if helper == "" {
		helper, helperErr := getCredentialHelper()
		if helperErr != nil {
			return creds, fmt.Errorf("get credential helper: %w", helperErr)
		}

		if helper == "" {
			return creds, nil
		}

		credHelperName = helper
	}

	helper = "docker-credential-" + credHelperName
	p, err := execLookPath(helper)
	if err != nil {
		if !errors.Is(err, exec.ErrNotFound) {
			return creds, fmt.Errorf("look up %q: %w", helper, err)
		}

		return creds, nil
	}

	var outBuf, errBuf bytes.Buffer
	cmd := execCommand(p, "get")
	cmd.Stdin = strings.NewReader(hostname)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if err = cmd.Run(); err != nil {
		out := strings.TrimSpace(outBuf.String())
		switch out {
		case ErrCredentialsNotFound.Error():
			return creds, nil
		case ErrCredentialsMissingServerURL.Error():
			return creds, ErrCredentialsMissingServerURL
		default:
			return creds, fmt.Errorf("execute %q stdout: %q stderr: %q: %w",
				helper, out, strings.TrimSpace(errBuf.String()), err,
			)
		}
	}

	// ServerURL is not always present in the output,
	// only some credential helpers include it (e.g. Google Cloud).
	var bytesCreds struct {
		Username  string `json:"Username"`
		Secret    string `json:"Secret"`
		ServerURL string `json:"ServerURL,omitempty"`
	}

	if err = json.Unmarshal(outBuf.Bytes(), &bytesCreds); err != nil {
		return creds, fmt.Errorf("unmarshal credentials from: %q: %w", helper, err)
	}

	// When tokenUsername is used, the output is an identity token and the username is garbage.
	if bytesCreds.Username == tokenUsername {
		bytesCreds.Username = ""
	}

	creds.Username = bytesCreds.Username
	creds.Password = bytesCreds.Secret
	creds.ServerAddress = bytesCreds.ServerURL

	return creds, nil
}

// getCredentialHelper gets the default credential helper name for the current platform.
func getCredentialHelper() (string, error) {
	switch runtime.GOOS {
	case "linux":
		if _, err := execLookPath("pass"); err != nil {
			if errors.Is(err, exec.ErrNotFound) {
				return "secretservice", nil
			}
			return "", fmt.Errorf(`look up "pass": %w`, err)
		}
		return "pass", nil
	case "darwin":
		return "osxkeychain", nil
	case "windows":
		return "wincred", nil
	default:
		return "", nil
	}
}
