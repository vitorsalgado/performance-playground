.ONESHELL:
.DEFAULT_GOAL := help

-include Makefile.overrides
-include .env

export

# Terminal styling helpers (ANSI escape codes).
reset := \033[0m
bold  := \033[1m
green := \033[32m

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

.PHONY: init
init: ## initialize the project with default configurations
	@time make gen-dsp-config
	@time make gen-apps
	@printf "\n"
	@printf "$(bold)%-18s$(reset) $(green)%s$(reset)\n" "Apps:" "d/apps.json"
	@printf "$(bold)%-18s$(reset) $(green)%s$(reset)\n" "DSPs:" "d/dsps.json"
	@printf "\n"

.PHONY: gen-apps
gen-apps: ## generate d/apps.json from .env APP_COUNT and APP_PUBLISHER_COUNT
	@node bin/gen-apps.js \
		--count 500000 \
		--publisher-count 500 \
		--start-id 1250 \
		--out d/apps.json

.PHONY: gen-dsp-config
gen-dsp-config: ## generate d/dsps.json and d/dsp-latencies.json from .env DSP_COUNT
	@node bin/gen-dsp-config.js

.PHONY: up
up: ## run everything
	@$(DOCKER_COMPOSE) up --detach --build --force-recreate

	@printf "\n"
	@printf "$(bold)%-18s$(reset) $(green)%s$(reset)\n" "Exchange LB:" "http://localhost:8080"
	@printf "$(bold)%-18s$(reset) $(green)%s$(reset)\n" "VictoriaMetrics:" "http://localhost:8428"
	@printf "$(bold)%-18s$(reset) $(green)%s$(reset)\n" "VictoriaLogs:" "http://localhost:9428/select/vmui"
	@printf "$(bold)%-18s$(reset) $(green)%s$(reset)\n" "VictoriaAlert:" "http://localhost:8880/vmalert"
	@printf "$(bold)%-18s$(reset) $(green)%s$(reset)\n" "Grafana:" "http://localhost:3000"
	@printf "$(bold)%-18s$(reset) $(green)%s$(reset)\n" "Alertmanager:" "http://localhost:9093"
	@printf "$(bold)%-18s$(reset) $(green)%s$(reset)\n" "Pyroscope:" "http://localhost:4040"
	@printf "\n"

.PHONY: up-attached
up-attached: ## run everything attached
	@$(DOCKER_COMPOSE) up --build --force-recreate

.PHONY: down
down: ## stop everything
	@$(DOCKER_COMPOSE) down --remove-orphans --volumes

.PHONY: run-exchange
run-exchange: ## run the exchange application
	EXCHANGE_APPS_CACHE_PATH=./d/apps.json \
	EXCHANGE_DSPS_CACHE_PATH=./d/dsps.json \
	go run ./exchange/exchange.go

.PHONY: run-dsp
run-dsp: ## run the dsp application (uses d/dsp-latencies.json if present, else latency 0)
	@DSP_LATENCIES_PATH=./d/dsp-latencies.json go run ./dsp/dsp.go

## --
## Testing
## --

.PHONY: testci
testci: ## run tests with a focus on ci
	@go test -v -race ./... -coverpkg=./... -coverprofile=coverage.txt

.PHONY: lint
lint: ## run linters
	@time golangci-lint run

.PHONY: k6
k6: ## run k6 load tests
	K6_WEB_DASHBOARD=true \
	K6_WEB_DASHBOARD_OPEN=true \
	K6_WEB_DASHBOARD_EXPORT=./load-testing/report.k6.html \
	k6 run --summary-export=./load-testing/summary.k6.json ./load-testing/k6.js

## --
## o11y
## --

.PHONY: logs-exchange
logs-exchange: ## show logs for the exchange service (follow mode)
	@$(DOCKER_COMPOSE) logs -f exchange

## --
## Setup
## --

.PHONY: deps
deps: ## dependencies
	@go mod download

## --
## Grafana
## --

GRAFANA_URL ?= http://localhost:3000
GRAFANA_ADMIN_USER ?= admin
GRAFANA_ADMIN_PASSWORD ?= admin

.PHONY: grafana-dashboards-reload
grafana-dashboards-reload: ## reload provisioned Grafana dashboards (no container restart)
	@curl -fsS \
		-X POST "$(GRAFANA_URL)/api/admin/provisioning/dashboards/reload" \
		-u "$(GRAFANA_ADMIN_USER):$(GRAFANA_ADMIN_PASSWORD)" \
		>/dev/null
	@printf "$(bold)%-24s$(reset) $(green)%s$(reset)\n" "Grafana dashboards:" "reloaded"

## -
## Misc
## --

.PHONY: help
help: ## show help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-24s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
