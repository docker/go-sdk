# Run make lint in all modules defined in go.work
lint-all:
	@go work edit -json | jq -r '.Use[].DiskPath' | while read -r module; do \
		echo "Running lint in $$module"; \
		(cd "$$module" && make lint) || exit 1; \
	done