// Package config provides legacy adapters for converting between go-sdk config types
// and Docker CLI/Engine API types.
//
// Deprecated: This package is deprecated and will be removed in a future release
// when all Docker products have migrated to use the go-sdk natively.
// Use the native go-sdk types directly and these adapters only when needed.
package config

import (
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/go-sdk/config"
)

// ToRegistryAuthConfig converts a go-sdk AuthConfig to Docker Engine API registry.AuthConfig.
//
// Deprecated: This function is deprecated and will be removed in a future release.
// Use the native go-sdk types directly and these adapters only when needed.
func ToRegistryAuthConfig(authConfig config.AuthConfig) registry.AuthConfig {
	return registry.AuthConfig{
		Username:      authConfig.Username,
		Password:      authConfig.Password,
		Auth:          authConfig.Auth,
		Email:         "",
		ServerAddress: authConfig.ServerAddress,
		IdentityToken: authConfig.IdentityToken,
		RegistryToken: authConfig.RegistryToken,
	}
}

// ToConfigFile converts a go-sdk Config to Docker CLI configfile.ConfigFile.
//
// Deprecated: This function is deprecated and will be removed in a future release.
// Use the native go-sdk types directly and these adapters only when needed.
func ToConfigFile(cfg config.Config) *configfile.ConfigFile {
	return &configfile.ConfigFile{
		AuthConfigs:          ToCLIAuthConfigs(cfg.AuthConfigs),
		HTTPHeaders:          cfg.HTTPHeaders,
		PsFormat:             cfg.PsFormat,
		ImagesFormat:         cfg.ImagesFormat,
		NetworksFormat:       cfg.NetworksFormat,
		PluginsFormat:        cfg.PluginsFormat,
		VolumesFormat:        cfg.VolumesFormat,
		StatsFormat:          cfg.StatsFormat,
		DetachKeys:           cfg.DetachKeys,
		CredentialsStore:     cfg.CredentialsStore,
		CredentialHelpers:    cfg.CredentialHelpers,
		Filename:             cfg.Filename,
		ServiceInspectFormat: cfg.ServiceInspectFormat,
		ServicesFormat:       cfg.ServicesFormat,
		TasksFormat:          cfg.TasksFormat,
		SecretFormat:         cfg.SecretFormat,
		ConfigFormat:         cfg.ConfigFormat,
		NodesFormat:          cfg.NodesFormat,
		PruneFilters:         cfg.PruneFilters,
		Proxies:              ToCLIProxyConfigs(cfg.Proxies),
		CurrentContext:       cfg.CurrentContext,
		CLIPluginsExtraDirs:  cfg.CLIPluginsExtraDirs,
		Plugins:              make(map[string]map[string]string),
		Aliases:              cfg.Aliases,
		Features:             make(map[string]string),
		Experimental:         cfg.Experimental,
	}
}

// ToCLIAuthConfig converts a go-sdk AuthConfig to Docker CLI types.AuthConfig.
// Note: RegistryToken field is included but may not be used by all CLI components.
//
// Deprecated: This function is deprecated and will be removed in a future release.
// Use the native go-sdk types directly and these adapters only when needed.
func ToCLIAuthConfig(authConfig config.AuthConfig) types.AuthConfig {
	return types.AuthConfig{
		Username:      authConfig.Username,
		Password:      authConfig.Password,
		Auth:          authConfig.Auth,
		Email:         "",
		ServerAddress: authConfig.ServerAddress,
		IdentityToken: authConfig.IdentityToken,
		RegistryToken: authConfig.RegistryToken,
	}
}

// ToCLIAuthConfigs converts a map of go-sdk AuthConfigs to Docker CLI types.AuthConfigs.
//
// Deprecated: This function is deprecated and will be removed in a future release.
// Use the native go-sdk types directly and these adapters only when needed.
func ToCLIAuthConfigs(authConfigs map[string]config.AuthConfig) map[string]types.AuthConfig {
	result := make(map[string]types.AuthConfig, len(authConfigs))
	for name, authConfig := range authConfigs {
		result[name] = ToCLIAuthConfig(authConfig)
	}
	return result
}

// ToCLIProxyConfig converts a go-sdk ProxyConfig to Docker CLI configfile.ProxyConfig.
//
// Deprecated: This function is deprecated and will be removed in a future release.
// Use the native go-sdk types directly and these adapters only when needed.
func ToCLIProxyConfig(proxy config.ProxyConfig) configfile.ProxyConfig {
	return configfile.ProxyConfig{
		HTTPProxy:  proxy.HTTPProxy,
		HTTPSProxy: proxy.HTTPSProxy,
		FTPProxy:   proxy.FTPProxy,
		NoProxy:    proxy.NoProxy,
	}
}

// ToCLIProxyConfigs converts a map of go-sdk ProxyConfigs to Docker CLI configfile.ProxyConfigs.
//
// Deprecated: This function is deprecated and will be removed in a future release.
// Use the native go-sdk types directly and these adapters only when needed.
func ToCLIProxyConfigs(proxies map[string]config.ProxyConfig) map[string]configfile.ProxyConfig {
	result := make(map[string]configfile.ProxyConfig, len(proxies))
	for name, proxy := range proxies {
		result[name] = ToCLIProxyConfig(proxy)
	}
	return result
}
