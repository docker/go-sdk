package config

import (
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/docker/docker/api/types/registry"
)

func TestDecodeBase64Auth(t *testing.T) {
	for _, tc := range base64TestCases() {
		t.Run(tc.name, testBase64Case(tc, func() (registry.AuthConfig, error) {
			user, pass, err := decodeBase64Auth(tc.config)
			return registry.AuthConfig{
				Username: user,
				Password: pass,
			}, err
		}))
	}
}

func TestConfig_RegistryCredentialsForHostname(t *testing.T) {
	t.Run("from base64 auth", func(t *testing.T) {
		for _, tc := range base64TestCases() {
			t.Run(tc.name, func(t *testing.T) {
				config := Config{
					AuthConfigs: map[string]registry.AuthConfig{
						"some.domain": tc.config,
					},
				}
				testBase64Case(tc, func() (registry.AuthConfig, error) {
					return config.AuthConfigForHostname("some.domain")
				})(t)
			})
		}
	})
}

type base64TestCase struct {
	name    string
	config  registry.AuthConfig
	expUser string
	expPass string
	expErr  bool
}

func base64TestCases() []base64TestCase {
	cases := []base64TestCase{
		{name: "empty"},
		{name: "not base64", expErr: true, config: registry.AuthConfig{Auth: "not base64"}},
		{name: "invalid format", expErr: true, config: registry.AuthConfig{
			Auth: base64.StdEncoding.EncodeToString([]byte("invalid format")),
		}},
		{name: "happy case", expUser: "user", expPass: "pass", config: registry.AuthConfig{
			Auth: base64.StdEncoding.EncodeToString([]byte("user:pass")),
		}},
	}

	return cases
}

type testAuthFn func() (registry.AuthConfig, error)

func testBase64Case(tc base64TestCase, authFn testAuthFn) func(t *testing.T) {
	return func(t *testing.T) {
		t.Helper()

		creds, err := authFn()
		if tc.expErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}

		require.Equal(t, tc.expUser, creds.Username)
		require.Equal(t, tc.expPass, creds.Password)
	}
}

// validateAuthForHostname is a helper function to validate the username and password for a given hostname.
func validateAuthForHostname(t *testing.T, hostname, expectedUser, expectedPass string, expectedErr error) {
	t.Helper()

	creds, err := AuthConfigForHostname(hostname)
	if expectedErr != nil {
		require.ErrorContains(t, err, expectedErr.Error())
	} else {
		require.NoError(t, err)
	}
	require.Equal(t, expectedUser, creds.Username)
	require.Equal(t, expectedPass, creds.Password)
	if creds.ServerAddress != "" {
		require.Equal(t, hostname, creds.ServerAddress)
	}
}

// validateAuthForImage is a helper function to validate the username and password for a given image reference.
func validateAuthForImage(t *testing.T, imageRef, expectedUser, expectedPass, expectedRegistry string, expectedErr error) {
	t.Helper()

	authConfigs, err := AuthConfigs(imageRef)
	if expectedErr != nil {
		require.ErrorContains(t, err, expectedErr.Error())
	} else {
		require.NoError(t, err)
	}

	creds, ok := authConfigs[expectedRegistry]
	require.Equal(t, (expectedErr == nil), ok)

	require.Equal(t, expectedUser, creds.Username)
	require.Equal(t, expectedPass, creds.Password)
	require.Equal(t, expectedRegistry, creds.ServerAddress)
}

// validateAuthErrorForHostname is a helper function to validate we get an error for the given hostname.
func validateAuthErrorForHostname(t *testing.T, hostname string, expectedErr error) {
	t.Helper()

	creds, err := AuthConfigForHostname(hostname)
	require.Error(t, err)
	require.Equal(t, expectedErr.Error(), err.Error())
	require.Empty(t, creds.Username)
	require.Empty(t, creds.Password)
	if creds.ServerAddress != "" {
		require.Equal(t, hostname, creds.ServerAddress)
	}
}

// validateAuthErrorForImage is a helper function to validate we get an error for the given image reference.
func validateAuthErrorForImage(t *testing.T, imageRef string, expectedErr error) {
	t.Helper()

	authConfigs, err := AuthConfigs(imageRef)
	require.Error(t, err)
	require.ErrorContains(t, err, expectedErr.Error())
	require.Empty(t, authConfigs)
}

func TestRegistryCredentialsForImage(t *testing.T) {
	t.Setenv(EnvOverrideDir, filepath.Join("testdata", "credhelpers-config"))

	t.Run("auths/user-pass", func(t *testing.T) {
		validateAuthForImage(t, "userpass.io/repo/image:tag", "user", "pass", "userpass.io", nil)
	})

	t.Run("auths/auth", func(t *testing.T) {
		validateAuthForImage(t, "auth.io/repo/image:tag", "auth", "authsecret", "auth.io", nil)
	})

	t.Run("credsStore", func(t *testing.T) {
		validateAuthForImage(t, "credstore.io/repo/image:tag", "", "", "credstore.io", nil)
	})

	t.Run("credHelpers/user-pass", func(t *testing.T) {
		mockExecCommand(t, `HELPER_STDOUT={"Username":"credhelper","Secret":"credhelpersecret", "ServerURL":"helper.io"}`)
		validateAuthForImage(t, "helper.io/repo/image:tag", "credhelper", "credhelpersecret", "helper.io", nil)
	})

	t.Run("credHelpers/token", func(t *testing.T) {
		mockExecCommand(t, `HELPER_STDOUT={"Username":"<token>", "Secret":"credhelpersecret", "ServerURL":"helper.io"}`)
		validateAuthForImage(t, "helper.io/repo/image:tag", "", "credhelpersecret", "helper.io", nil)
	})

	t.Run("credHelpers/not-found", func(t *testing.T) {
		mockExecCommand(t, "HELPER_STDOUT="+ErrCredentialsNotFound.Error(), "HELPER_EXIT_CODE=1")
		validateAuthForImage(t, "helper.io/repo/image:tag", "", "", "helper.io", nil)
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
		validateAuthForImage(t, "other.io/repo/image:tag", "", "", "other.io", nil)
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
		validateAuthForImage(t, "userpass.io/repo/image:tag", "", "", "", errors.New("file does not exist"))
	})
}

func TestRegistryCredentialsForHostname(t *testing.T) {
	t.Setenv(EnvOverrideDir, filepath.Join("testdata", "credhelpers-config"))

	t.Run("auths/user-pass", func(t *testing.T) {
		validateAuthForHostname(t, "userpass.io", "user", "pass", nil)
	})

	t.Run("auths/auth", func(t *testing.T) {
		validateAuthForHostname(t, "auth.io", "auth", "authsecret", nil)
	})

	t.Run("credsStore", func(t *testing.T) {
		validateAuthForHostname(t, "credstore.io", "", "", nil)
	})

	t.Run("credHelpers/user-pass", func(t *testing.T) {
		mockExecCommand(t, `HELPER_STDOUT={"Username":"credhelper","Secret":"credhelpersecret"}`)
		validateAuthForHostname(t, "helper.io", "credhelper", "credhelpersecret", nil)
	})

	t.Run("credHelpers/token", func(t *testing.T) {
		mockExecCommand(t, `HELPER_STDOUT={"Username":"<token>", "Secret":"credhelpersecret"}`)
		validateAuthForHostname(t, "helper.io", "", "credhelpersecret", nil)
	})

	t.Run("credHelpers/not-found", func(t *testing.T) {
		mockExecCommand(t, "HELPER_STDOUT="+ErrCredentialsNotFound.Error(), "HELPER_EXIT_CODE=1")
		validateAuthForHostname(t, "helper.io", "", "", nil)
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
		validateAuthForHostname(t, "other.io", "", "", nil)
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
		validateAuthForHostname(t, "userpass.io", "", "", errors.New("file does not exist"))
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

func TestAuthConfigs(t *testing.T) {
	t.Setenv(EnvOverrideDir, filepath.Join("testdata", "credhelpers-config"))

	t.Run("success", func(t *testing.T) {
		authConfigs, err := AuthConfigs("userpass.io/repo/image:tag")
		require.NoError(t, err)
		require.Equal(t, "user", authConfigs["userpass.io"].Username)
		require.Equal(t, "pass", authConfigs["userpass.io"].Password)
		require.Equal(t, "userpass.io", authConfigs["userpass.io"].ServerAddress)
	})

	t.Run("not-cached", func(t *testing.T) {
		authConfigs, err := AuthConfigs("notcached.io/repo/image:tag")
		require.NoError(t, err)
		require.Empty(t, authConfigs["notcached.io"].Username)
		require.Empty(t, authConfigs["notcached.io"].Password)
		require.Equal(t, "notcached.io", authConfigs["notcached.io"].ServerAddress)
	})
}
