.ONESHELL:
.DEFAULT_GOAL := help

# Allow user specific optional overrides
-include Makefile.overrides

export

# Prefer "docker compose" (plugin) but support legacy "docker-compose".
# Can be overridden via env/CLI, e.g. `make up DOCKER_COMPOSE=docker-compose`.
DOCKER_COMPOSE ?= $(shell \
	if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then \
		echo "docker compose"; \
	elif command -v docker-compose >/dev/null 2>&1; then \
		echo "docker-compose"; \
	else \
		echo "docker compose"; \
	fi \
)

.PHONY: up
up: ## run everything
	@$(DOCKER_COMPOSE) up --build --force-recreate

.PHONY: down
down: ## stop everything
	@$(DOCKER_COMPOSE) down --volumes --remove-orphans

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

.PHONY: k6
k6: ## run k6 load tests
	K6_WEB_DASHBOARD=true \
	K6_WEB_DASHBOARD_OPEN=true \
	K6_WEB_DASHBOARD_EXPORT=./load-testing/report.k6.html \
	k6 run --summary-export=./load-testing/summary.k6.json ./load-testing/index.js

## -
## Misc
## --

.PHONY: help
help: ## show help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-24s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
