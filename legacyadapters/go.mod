module github.com/docker/go-sdk/legacyadapters

go 1.23.6

replace github.com/docker/go-sdk/config => ../config

require (
	github.com/docker/cli v28.3.2+incompatible
	github.com/docker/docker v28.3.2+incompatible
	github.com/docker/go-sdk/config v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.10.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/docker-credential-helpers v0.9.3 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	golang.org/x/sys v0.20.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
