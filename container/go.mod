module github.com/docker/go-sdk/container

go 1.23.6

replace (
	github.com/docker/go-sdk/client => ../client
	github.com/docker/go-sdk/config => ../config
	github.com/docker/go-sdk/context => ../context
	github.com/docker/go-sdk/image => ../image
	github.com/docker/go-sdk/network => ../network
)

require (
	dario.cat/mergo v1.0.2
	github.com/containerd/errdefs v1.0.0
	github.com/containerd/platforms v0.2.1
	github.com/docker/docker v28.3.2+incompatible
	github.com/docker/go-connections v0.5.0
	github.com/docker/go-sdk/client v0.1.0-alpha010
	github.com/docker/go-sdk/config v0.1.0-alpha010
	github.com/docker/go-sdk/image v0.1.0-alpha011
	github.com/docker/go-sdk/network v0.1.0-alpha010
	github.com/stretchr/testify v1.10.0
	golang.org/x/sys v0.33.0
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/caarlos0/env/v11 v11.3.1 // indirect
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-sdk/context v0.1.0-alpha010 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/go-archive v0.1.0 // indirect
	github.com/moby/patternmatcher v0.6.0 // indirect
	github.com/moby/sys/sequential v0.6.0 // indirect
	github.com/moby/sys/user v0.4.0 // indirect
	github.com/moby/sys/userns v0.1.0 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.61.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
