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

clean-build-dir:
	@echo "Cleaning build directory..."
	@rm -rf .github/scripts/.build
	@mkdir -p .github/scripts/.build

# Pre-release version for all modules
pre-release-all: clean-build-dir
	@echo "Preparing releasing versions for all modules..."
	$(call for-all-modules,make pre-release)

# Release version for all modules. It must be run after pre-release-all.
release-all:
	$(call for-all-modules,make check-pre-release)
	@./.github/scripts/release.sh
