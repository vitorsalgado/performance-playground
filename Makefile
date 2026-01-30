.ONESHELL:
.DEFAULT_GOAL := help

# Allow user specific optional overrides
-include Makefile.overrides

export

.PHONY: up
up: ## run everything
	@docker-compose up --build --force-recreate

.PHONY: down
down: ## stop everything
	@docker-compose down --volumes --remove-orphans

.PHONY: run
run: ## run the application
	@go run ./cmd/exchange

## --
## Testing
## --

.PHONY: testci
testci: ## run tests with a focus on ci
	@go test -v -race ./... -coverpkg=./... -coverprofile=coverage.txt

.PHONY: lint
lint: ## run linters
	@time golangci-lint run

.PHONY: deps
deps: ## dependencies
	@go mod download

## -
## Misc
## --

.PHONY: help
help: ## show help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-24s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
