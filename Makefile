# Function to execute a command in all modules
define for-all-modules
	@go work edit -json | jq -r '.Use[].DiskPath' | while read -r module; do \
		echo "Processing module: $$module"; \
		(cd "$$module" && $(1)) || exit 1; \
	done
endef

# Run make lint in all modules defined in go.work
lint-all:
	@echo "Running lint in all modules..."
	$(call for-all-modules,make lint)

tidy-all:
	@echo "Running tidy in all modules..."
	$(call for-all-modules,go mod tidy)

# Release alpha version for all modules
release-alpha-all:
	@echo "Releasing alpha versions for all modules..."
	$(call for-all-modules,../.github/scripts/release-alpha.sh v0.1.0 "$$(basename "$$module")")

# Release alpha version for a specific module
release-alpha:
	@if [ -z "$(MODULE)" ]; then \
		echo "Usage: make release-alpha MODULE=<module-name>"; \
		echo "Available modules: client, config, container, context, image, network"; \
		exit 1; \
	fi
	@echo "Releasing alpha version for module: $(MODULE)"
	@cd "$(MODULE)" && ../.github/scripts/release-alpha.sh v0.1.0 "$(MODULE)"
