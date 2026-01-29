.ONESHELL:
.DEFAULT_GOAL := help

# Allow user specific optional overrides
-include Makefile.overrides

export

## -
## Misc
## --

.PHONY: help
help: ## show help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-24s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
