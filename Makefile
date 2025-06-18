# Run make lint in all modules defined in go.work
lint-all:
	@go work edit -json | jq -r '.Use[].DiskPath' | while read -r module; do \
		echo "Running lint in $$module"; \
		(cd "$$module" && make lint) || exit 1; \
	done

tidy-all:
	@go work edit -json | jq -r '.Use[].DiskPath' | while read -r module; do \
		echo "Running tidy in $$module"; \
		(cd "$$module" && go mod tidy) || exit 1; \
	done

# Release alpha version for all modules
release-alpha-all:
	@echo "Releasing alpha versions for all modules..."
	@go work edit -json | jq -r '.Use[].DiskPath' | while read -r module; do \
		echo "Processing module: $$module"; \
		(cd "$$module" && ../.github/scripts/release-alpha.sh v0.1.0 "$$(basename "$$module")") || exit 1; \
	done

# Release alpha version for a specific module
release-alpha:
	@if [ -z "$(MODULE)" ]; then \
		echo "Usage: make release-alpha MODULE=<module-name>"; \
		echo "Available modules: client, config, container, context, image, network"; \
		exit 1; \
	fi
	@echo "Releasing alpha version for module: $(MODULE)"
	@cd "$(MODULE)" && ../.github/scripts/release-alpha.sh v0.1.0 "$(MODULE)"
