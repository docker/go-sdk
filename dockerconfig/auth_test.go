package dockerconfig

import (
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeBase64Auth(t *testing.T) {
	for _, tc := range base64TestCases() {
		t.Run(tc.name, testBase64Case(tc, func() (string, string, error) {
			return decodeBase64Auth(tc.config)
		}))
	}
}

func TestConfig_RegistryCredentialsForHostname(t *testing.T) {
	t.Run("from base64 auth", func(t *testing.T) {
		for _, tc := range base64TestCases() {
			t.Run(tc.name, func(t *testing.T) {
				config := Config{
					AuthConfigs: map[string]AuthConfig{
						"some.domain": tc.config,
					},
				}
				testBase64Case(tc, func() (string, string, error) {
					return config.RegistryCredentialsForHostname("some.domain")
				})(t)
			})
		}
	})
}

type base64TestCase struct {
	name    string
	config  AuthConfig
	expUser string
	expPass string
	expErr  bool
}

func base64TestCases() []base64TestCase {
	cases := []base64TestCase{
		{name: "empty"},
		{name: "not base64", expErr: true, config: AuthConfig{Auth: "not base64"}},
		{name: "invalid format", expErr: true, config: AuthConfig{
			Auth: base64.StdEncoding.EncodeToString([]byte("invalid format")),
		}},
		{name: "happy case", expUser: "user", expPass: "pass", config: AuthConfig{
			Auth: base64.StdEncoding.EncodeToString([]byte("user:pass")),
		}},
	}

	return cases
}

type testAuthFn func() (string, string, error)

func testBase64Case(tc base64TestCase, authFn testAuthFn) func(t *testing.T) {
	return func(t *testing.T) {
		t.Helper()

		u, p, err := authFn()
		if tc.expErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}

		require.Equal(t, tc.expUser, u)
		require.Equal(t, tc.expPass, p)
	}
}

// validateAuthForHostname is a helper function to validate the username and password for a given hostname.
func validateAuthForHostname(t *testing.T, hostname, expectedUser, expectedPass string) {
	t.Helper()

	username, password, err := RegistryCredentialsForHostname(hostname)
	require.NoError(t, err)
	require.Equal(t, expectedUser, username)
	require.Equal(t, expectedPass, password)
}

// validateAuthForImage is a helper function to validate the username and password for a given image reference.
func validateAuthForImage(t *testing.T, imageRef, expectedUser, expectedPass string) {
	t.Helper()

	username, password, err := RegistryCredentials(imageRef)
	require.NoError(t, err)
	require.Equal(t, expectedUser, username)
	require.Equal(t, expectedPass, password)
}

// validateAuthErrorForHostname is a helper function to validate we get an error for the given hostname.
func validateAuthErrorForHostname(t *testing.T, hostname string, expectedErr error) {
	t.Helper()

	username, password, err := RegistryCredentialsForHostname(hostname)
	require.Error(t, err)
	require.Equal(t, expectedErr.Error(), err.Error())
	require.Empty(t, username)
	require.Empty(t, password)
}

// validateAuthErrorForImage is a helper function to validate we get an error for the given image reference.
func validateAuthErrorForImage(t *testing.T, imageRef string, expectedErr error) {
	t.Helper()

	username, password, err := RegistryCredentials(imageRef)
	require.Error(t, err)
	require.Equal(t, expectedErr.Error(), err.Error())
	require.Empty(t, username)
	require.Empty(t, password)
}

func TestRegistryCredentialsForImage(t *testing.T) {
	t.Setenv(EnvOverrideDir, filepath.Join("testdata", "credhelpers-config"))

	t.Run("auths/user-pass", func(t *testing.T) {
		validateAuthForImage(t, "userpass.io/repo/image:tag", "user", "pass")
	})

	t.Run("auths/auth", func(t *testing.T) {
		validateAuthForImage(t, "auth.io/repo/image:tag", "auth", "authsecret")
	})

	t.Run("credsStore", func(t *testing.T) {
		validateAuthForImage(t, "credstore.io/repo/image:tag", "", "")
	})

	t.Run("credHelpers/user-pass", func(t *testing.T) {
		mockExecCommand(t, `HELPER_STDOUT={"Username":"credhelper","Secret":"credhelpersecret"}`)
		validateAuthForImage(t, "helper.io/repo/image:tag", "credhelper", "credhelpersecret")
	})

	t.Run("credHelpers/token", func(t *testing.T) {
		mockExecCommand(t, `HELPER_STDOUT={"Username":"<token>", "Secret":"credhelpersecret"}`)
		validateAuthForImage(t, "helper.io/repo/image:tag", "", "credhelpersecret")
	})

	t.Run("credHelpers/not-found", func(t *testing.T) {
		mockExecCommand(t, "HELPER_STDOUT="+ErrCredentialsNotFound.Error(), "HELPER_EXIT_CODE=1")
		validateAuthForImage(t, "helper.io/repo/image:tag", "", "")
	})

	t.Run("credHelpers/missing-url", func(t *testing.T) {
		mockExecCommand(t, "HELPER_STDOUT="+ErrCredentialsMissingServerURL.Error(), "HELPER_EXIT_CODE=1")
		validateAuthErrorForImage(t, "helper.io/repo/image:tag", ErrCredentialsMissingServerURL)
	})

	t.Run("credHelpers/other-error", func(t *testing.T) {
		mockExecCommand(t, "HELPER_STDOUT=output", "HELPER_STDERR=my error", "HELPER_EXIT_CODE=10")
		expectedErr := errors.New(`execute "docker-credential-helper" stdout: "output" stderr: "my error": exit status 10`)
		validateAuthErrorForImage(t, "helper.io/repo/image:tag", expectedErr)
	})

	t.Run("credHelpers/lookup-not-found", func(t *testing.T) {
		mockExecCommand(t, "HELPER_STDOUT=output", "HELPER_STDERR=my error", "HELPER_EXIT_CODE=10")
		validateAuthForImage(t, "other.io/repo/image:tag", "", "")
	})

	t.Run("credHelpers/lookup-error", func(t *testing.T) {
		mockExecCommand(t, "HELPER_STDOUT=output", "HELPER_STDERR=my error", "HELPER_EXIT_CODE=10")
		expectedErr := errors.New(`look up "docker-credential-error": lookup error`)
		validateAuthErrorForImage(t, "error.io/repo/image:tag", expectedErr)
	})

	t.Run("credHelpers/decode-json", func(t *testing.T) {
		mockExecCommand(t, "HELPER_STDOUT=bad-json")
		expectedErr := errors.New(`unmarshal credentials from: "docker-credential-helper": invalid character 'b' looking for beginning of value`)
		validateAuthErrorForImage(t, "helper.io/repo/image:tag", expectedErr)
	})

	t.Run("config/not-found", func(t *testing.T) {
		t.Setenv(EnvOverrideDir, filepath.Join("testdata", "missing"))
		validateAuthForImage(t, "userpass.io/repo/image:tag", "", "")
	})
}

func TestRegistryCredentialsForHostname(t *testing.T) {
	t.Setenv(EnvOverrideDir, filepath.Join("testdata", "credhelpers-config"))

	t.Run("auths/user-pass", func(t *testing.T) {
		validateAuthForHostname(t, "userpass.io", "user", "pass")
	})

	t.Run("auths/auth", func(t *testing.T) {
		validateAuthForHostname(t, "auth.io", "auth", "authsecret")
	})

	t.Run("credsStore", func(t *testing.T) {
		validateAuthForHostname(t, "credstore.io", "", "")
	})

	t.Run("credHelpers/user-pass", func(t *testing.T) {
		mockExecCommand(t, `HELPER_STDOUT={"Username":"credhelper","Secret":"credhelpersecret"}`)
		validateAuthForHostname(t, "helper.io", "credhelper", "credhelpersecret")
	})

	t.Run("credHelpers/token", func(t *testing.T) {
		mockExecCommand(t, `HELPER_STDOUT={"Username":"<token>", "Secret":"credhelpersecret"}`)
		validateAuthForHostname(t, "helper.io", "", "credhelpersecret")
	})

	t.Run("credHelpers/not-found", func(t *testing.T) {
		mockExecCommand(t, "HELPER_STDOUT="+ErrCredentialsNotFound.Error(), "HELPER_EXIT_CODE=1")
		validateAuthForHostname(t, "helper.io", "", "")
	})

	t.Run("credHelpers/missing-url", func(t *testing.T) {
		mockExecCommand(t, "HELPER_STDOUT="+ErrCredentialsMissingServerURL.Error(), "HELPER_EXIT_CODE=1")
		validateAuthErrorForHostname(t, "helper.io", ErrCredentialsMissingServerURL)
	})

	t.Run("credHelpers/other-error", func(t *testing.T) {
		mockExecCommand(t, "HELPER_STDOUT=output", "HELPER_STDERR=my error", "HELPER_EXIT_CODE=10")
		expectedErr := errors.New(`execute "docker-credential-helper" stdout: "output" stderr: "my error": exit status 10`)
		validateAuthErrorForHostname(t, "helper.io", expectedErr)
	})

	t.Run("credHelpers/lookup-not-found", func(t *testing.T) {
		mockExecCommand(t, "HELPER_STDOUT=output", "HELPER_STDERR=my error", "HELPER_EXIT_CODE=10")
		validateAuthForHostname(t, "other.io", "", "")
	})

	t.Run("credHelpers/lookup-error", func(t *testing.T) {
		mockExecCommand(t, "HELPER_STDOUT=output", "HELPER_STDERR=my error", "HELPER_EXIT_CODE=10")
		expectedErr := errors.New(`look up "docker-credential-error": lookup error`)
		validateAuthErrorForHostname(t, "error.io", expectedErr)
	})

	t.Run("credHelpers/decode-json", func(t *testing.T) {
		mockExecCommand(t, "HELPER_STDOUT=bad-json")
		expectedErr := errors.New(`unmarshal credentials from: "docker-credential-helper": invalid character 'b' looking for beginning of value`)
		validateAuthErrorForHostname(t, "helper.io", expectedErr)
	})

	t.Run("config/not-found", func(t *testing.T) {
		t.Setenv(EnvOverrideDir, filepath.Join("testdata", "missing"))
		validateAuthForHostname(t, "userpass.io", "", "")
	})
}

// TestMain is hijacked so we can run a test helper which can write
// cleanly to stdout and stderr.
func TestMain(m *testing.M) {
	pid := os.Getpid()
	if os.Getenv("GO_EXEC_TEST_PID") == "" {
		os.Setenv("GO_EXEC_TEST_PID", strconv.Itoa(pid))
		// Run the tests.
		os.Exit(m.Run())
	}

	// Run the helper which slurps stdin and writes to stdout and stderr.
	if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
		if _, err = os.Stderr.WriteString(err.Error()); err != nil {
			panic(err)
		}
	}

	if out := os.Getenv("HELPER_STDOUT"); out != "" {
		if _, err := os.Stdout.WriteString(out); err != nil {
			panic(err)
		}
	}

	if out := os.Getenv("HELPER_STDERR"); out != "" {
		if _, err := os.Stderr.WriteString(out); err != nil {
			panic(err)
		}
	}

	if code := os.Getenv("HELPER_EXIT_CODE"); code != "" {
		code, err := strconv.Atoi(code)
		if err != nil {
			panic(err)
		}

		os.Exit(code)
	}
}
