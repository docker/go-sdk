package config

import "sync/atomic"

const (
	// EnvOverrideDir is the name of the environment variable that can be
	// used to override the location of the client configuration files (~/.docker).
	//
	// It takes priority over the default.
	EnvOverrideDir = "DOCKER_CONFIG"

	// configFileDir is the name of the directory containing the client configuration files
	configFileDir = ".docker"

	// configFileName is the name of the client configuration file inside the
	// config-directory.
	FileName = "config.json"
)

// Config represents the on disk format of the docker CLI's config file.
type Config struct {
	filepath             string                 `json:"-"`
	AuthConfigs          map[string]AuthConfig  `json:"auths"`
	HTTPHeaders          map[string]string      `json:"HttpHeaders,omitempty"`
	PsFormat             string                 `json:"psFormat,omitempty"`
	ImagesFormat         string                 `json:"imagesFormat,omitempty"`
	NetworksFormat       string                 `json:"networksFormat,omitempty"`
	PluginsFormat        string                 `json:"pluginsFormat,omitempty"`
	VolumesFormat        string                 `json:"volumesFormat,omitempty"`
	StatsFormat          string                 `json:"statsFormat,omitempty"`
	DetachKeys           string                 `json:"detachKeys,omitempty"`
	CredentialsStore     string                 `json:"credsStore,omitempty"`
	CredentialHelpers    map[string]string      `json:"credHelpers,omitempty"`
	Filename             string                 `json:"-"` // Note: for internal use only.
	ServiceInspectFormat string                 `json:"serviceInspectFormat,omitempty"`
	ServicesFormat       string                 `json:"servicesFormat,omitempty"`
	TasksFormat          string                 `json:"tasksFormat,omitempty"`
	SecretFormat         string                 `json:"secretFormat,omitempty"`
	ConfigFormat         string                 `json:"configFormat,omitempty"`
	NodesFormat          string                 `json:"nodesFormat,omitempty"`
	PruneFilters         []string               `json:"pruneFilters,omitempty"`
	Proxies              map[string]ProxyConfig `json:"proxies,omitempty"`
	Experimental         string                 `json:"experimental,omitempty"`
	StackOrchestrator    string                 `json:"stackOrchestrator,omitempty"`
	Kubernetes           *KubernetesConfig      `json:"kubernetes,omitempty"`
	CurrentContext       string                 `json:"currentContext,omitempty"`
	CLIPluginsExtraDirs  []string               `json:"cliPluginsExtraDirs,omitempty"`
	Aliases              map[string]string      `json:"aliases,omitempty"`

	// Cache pointer (unexported, not included in JSON, safe to copy)
	cache atomic.Value // stores *authConfigCache
}

// ProxyConfig contains proxy configuration settings.
type ProxyConfig struct {
	HTTPProxy  string `json:"httpProxy,omitempty"`
	HTTPSProxy string `json:"httpsProxy,omitempty"`
	NoProxy    string `json:"noProxy,omitempty"`
	FTPProxy   string `json:"ftpProxy,omitempty"`
	AllProxy   string `json:"allProxy,omitempty"`
}

// AuthConfig contains authorization information for connecting to a Registry.
type AuthConfig struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Auth          string `json:"auth,omitempty"`
	ServerAddress string `json:"serveraddress,omitempty"`

	// IdentityToken is used to authenticate the user and get
	// an access token for the registry.
	IdentityToken string `json:"identitytoken,omitempty"`

	// RegistryToken is a bearer token to be sent to a registry.
	RegistryToken string `json:"registrytoken,omitempty"`
}

// KubernetesConfig contains Kubernetes orchestrator settings.
type KubernetesConfig struct {
	AllNamespaces string `json:"allNamespaces,omitempty"`
}
