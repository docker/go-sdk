ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
MODULE_DIR:=$(shell basename $(CURDIR))
GOBIN= $(GOPATH)/bin

# ------------------------------------------------------------------------------
# Install Tools
# ------------------------------------------------------------------------------

define go_install
    go install $(1)
endef

$(GOBIN)/golangci-lint:
	$(call go_install,github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2)

$(GOBIN)/gotestsum:
	$(call go_install,gotest.tools/gotestsum@latest)

$(GOBIN)/mockery:
	$(call go_install,github.com/vektra/mockery/v2@v2.53.4)

.PHONY: install-tools
install-tools: $(GOBIN)/golangci-lint $(GOBIN)/gotestsum $(GOBIN)/mockery

.PHONY: clean-tools
clean-tools:
	rm $(GOBIN)/golangci-lint
	rm $(GOBIN)/gotestsum
	rm $(GOBIN)/mockery

# ------------------------------------------------------------------------------
# Test
# ------------------------------------------------------------------------------

.PHONY: test
test: $(GOBIN)/gotestsum
	@echo "Running tests..."
	gotestsum \
		--format short-verbose \
		--packages="./..." \
		--junitfile TEST-unit.xml \
		-- \
		-v \
		-coverprofile=coverage.out \
		-timeout=30m \
		-race

# ------------------------------------------------------------------------------
# Static code analysis
# ------------------------------------------------------------------------------
.PHONY: pre-commit
pre-commit: tidy lint

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: lint
lint: $(GOBIN)/golangci-lint
	golangci-lint run --verbose -c $(ROOT_DIR)/.golangci.yml --fix

# ------------------------------------------------------------------------------
# Mockery
# ------------------------------------------------------------------------------
.PHONY: generate
generate: $(GOBIN)/mockery
	go generate ./...

# ------------------------------------------------------------------------------
# Security
# ------------------------------------------------------------------------------
.PHONY: dependencies-scan
dependencies-scan:
	@echo ">> Scanning dependencies in $(CURDIR)..."
	go list -json -m all | docker run --rm -i sonatypecommunity/nancy:latest sleuth --skip-update-check

# ------------------------------------------------------------------------------
# Release
# ------------------------------------------------------------------------------
.PHONY: tag-release
tag-release:
	@if [ -z "$(MODULE_DIR)" ]; then \
		echo "Usage: make tag-release, from one of the module directories (e.g. make tag-release from client/ directory)"; \
		exit 1; \
	fi
	@if [ ! -f "$(ROOT_DIR)/$(MODULE_DIR)/version.go" ]; then \
		echo "Error: $(MODULE_DIR)/version.go not found. Is this a valid module directory?"; \
		exit 1; \
	fi
	@$(ROOT_DIR)/.github/scripts/tag-release.sh "$(MODULE_DIR)"

.PHONY: refresh-proxy
refresh-proxy:
	@if [ -z "$(MODULE_DIR)" ]; then \
		echo "Usage: make refresh-proxy, from one of the module directories (e.g. make refresh-proxy from client/ directory)"; \
		exit 1; \
	fi
	@$(ROOT_DIR)/.github/scripts/refresh-proxy.sh "$(MODULE_DIR)"
