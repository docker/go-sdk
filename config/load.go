package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

// getHomeDir returns the home directory of the current user with the help of
// environment variables depending on the target operating system.
// Returned path should be used with "path/filepath" to form new paths.
//
// On non-Windows platforms, it falls back to nss lookups, if the home
// directory cannot be obtained from environment-variables.
//
// If linking statically with cgo enabled against glibc, ensure the
// osusergo build tag is used.
//
// If needing to do nss lookups, do not disable cgo or set osusergo.
//
// getHomeDir is a copy of [pkg/homedir.Get] to prevent adding docker/docker
// as dependency for consumers that only need to read the config-file.
//
// [pkg/homedir.Get]: https://pkg.go.dev/github.com/docker/docker@v26.1.4+incompatible/pkg/homedir#Get
func getHomeDir() (string, error) {
	home, _ := os.UserHomeDir()
	if home == "" && runtime.GOOS != "windows" {
		if u, err := user.Current(); err == nil {
			return u.HomeDir, nil
		}
	}

	if home == "" {
		return "", errors.New("user home directory not determined")
	}

	return home, nil
}

// Dir returns the directory the configuration file is stored in,
// checking if the directory exists.
//
// When the directory does not exist the returned error wraps
// [os.ErrNotExist] so that callers can detect the condition with
// [errors.Is] or [os.IsNotExist].
func Dir() (string, error) {
	dir := os.Getenv(EnvOverrideDir)
	if dir != "" {
		if _, err := os.Stat(dir); err != nil {
			return "", fmt.Errorf("config dir %s: %w", dir, err)
		}
		return dir, nil
	}

	home, err := getHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home dir: %w", err)
	}

	configDir := filepath.Join(home, configFileDir)
	if _, err := os.Stat(configDir); err != nil {
		return "", fmt.Errorf("config dir %s: %w", configDir, err)
	}

	return configDir, nil
}

// Filepath returns the path to the docker cli config file,
// checking if the file exists.
//
// When the directory or file does not exist the returned error wraps
// [os.ErrNotExist] so that callers can detect the condition with
// [errors.Is] or [os.IsNotExist].
func Filepath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", fmt.Errorf("config dir: %w", err)
	}

	configFilePath := filepath.Join(dir, FileName)
	if _, err := os.Stat(configFilePath); err != nil {
		return "", fmt.Errorf("config file %s: %w", configFilePath, err)
	}

	return configFilePath, nil
}

// Load returns the docker config file. It will internally check, in this particular order:
// 1. the DOCKER_AUTH_CONFIG environment variable, unmarshalling it into a Config
// 2. the DOCKER_CONFIG environment variable, as the path to the config file
// 3. else it will load the default config file, which is ~/.docker/config.json
//
// If the configuration directory or file does not exist, Load returns an
// empty [Config] and a nil error.  This matches the behaviour of
// [github.com/docker/cli/cli/config.Load] and allows callers to work on
// systems where Docker has never been installed.
func Load() (Config, error) {
	if env := os.Getenv("DOCKER_AUTH_CONFIG"); env != "" {
		var cfg Config
		if err := json.Unmarshal([]byte(env), &cfg); err != nil {
			return Config{}, fmt.Errorf("unmarshal DOCKER_AUTH_CONFIG: %w", err)
		}
		return cfg, nil
	}

	var cfg Config
	p, err := Filepath()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("config path: %w", err)
	}

	cfg, err = loadFromFilepath(p)
	if err != nil {
		return cfg, fmt.Errorf("load config: %w", err)
	}

	// store the location of the config file into the config, for future use
	cfg.filepath = p

	return cfg, nil
}

// loadFromFilepath loads config from the specified path into cfg.
func loadFromFilepath(configPath string) (Config, error) {
	var cfg Config
	f, err := os.Open(configPath)
	if err != nil {
		return cfg, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("decode config: %w", err)
	}

	return cfg, nil
}
