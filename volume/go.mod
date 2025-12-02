module github.com/docker/go-sdk/volume

go 1.24.0

replace (
	github.com/docker/go-sdk/client => ../client
	github.com/docker/go-sdk/config => ../config
	github.com/docker/go-sdk/context => ../context
)

require (
	github.com/containerd/errdefs v1.0.0
	github.com/docker/go-sdk/client v0.1.0-alpha011
	github.com/moby/moby/api v1.52.0
	github.com/moby/moby/client v0.1.0
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/caarlos0/env/v11 v11.3.1 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.6.0 // indirect
	github.com/docker/go-sdk/config v0.1.0-alpha011 // indirect
	github.com/docker/go-sdk/context v0.1.0-alpha011 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.61.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
