ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
GOBIN= $(GOPATH)/bin

<<<<<<< HEAD
# ------------------------------------------------------------------------------
# Install Tools
# ------------------------------------------------------------------------------

=======
>>>>>>> tcgo/main
define go_install
    go install $(1)
endef

$(GOBIN)/golangci-lint:
	$(call go_install,github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.0.2)

$(GOBIN)/gotestsum:
	$(call go_install,gotest.tools/gotestsum@latest)

$(GOBIN)/mockery:
	$(call go_install,github.com/vektra/mockery/v2@v2.45)

<<<<<<< HEAD
.PHONY: install-tools
install-tools: $(GOBIN)/golangci-lint $(GOBIN)/gotestsum $(GOBIN)/mockery

.PHONY: clean-tools
clean-tools:
=======
.PHONY: install
install: $(GOBIN)/golangci-lint $(GOBIN)/gotestsum $(GOBIN)/mockery

.PHONY: clean
clean:
>>>>>>> tcgo/main
	rm $(GOBIN)/golangci-lint
	rm $(GOBIN)/gotestsum
	rm $(GOBIN)/mockery

<<<<<<< HEAD
# ------------------------------------------------------------------------------
# Test
# ------------------------------------------------------------------------------

.PHONY: test
test: $(GOBIN)/gotestsum
	@echo "Running tests..."
=======
.PHONY: dependencies-scan
dependencies-scan:
	@echo ">> Scanning dependencies in $(CURDIR)..."
	go list -json -m all | docker run --rm -i sonatypecommunity/nancy:latest sleuth --skip-update-check

.PHONY: lint
lint: $(GOBIN)/golangci-lint
	golangci-lint run --verbose -c $(ROOT_DIR)/.golangci.yml --fix

.PHONY: generate
generate: $(GOBIN)/mockery
	go generate ./...

.PHONY: test-%
test-%: $(GOBIN)/gotestsum
	@echo "Running $* tests..."
>>>>>>> tcgo/main
	gotestsum \
		--format short-verbose \
		--rerun-fails=5 \
		--packages="./..." \
		--junitfile TEST-unit.xml \
		-- \
		-v \
		-coverprofile=coverage.out \
		-timeout=30m \
		-race

<<<<<<< HEAD
# ------------------------------------------------------------------------------
# Static code analysis
# ------------------------------------------------------------------------------
.PHONY: pre-commit
pre-commit: tidy lint
=======
.PHONY: tools
tools:
	go mod download

.PHONY: test-tools
test-tools: $(GOBIN)/gotestsum
>>>>>>> tcgo/main

.PHONY: tidy
tidy:
	go mod tidy

<<<<<<< HEAD
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
=======
.PHONY: pre-commit
pre-commit: generate tidy lint
>>>>>>> tcgo/main
