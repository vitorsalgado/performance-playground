# AGENTS.md

Guidance for AI agents working on this repository.

## Project summary

Performance playground: a simulated ad-exchange–style system used to run load tests, benchmarks, and observability experiments. The app and stack are kept simple on purpose; some known inefficiencies are intentional so that changes show up clearly in metrics.

## Layout

- **`flavors/adtech/`** — Go applications: `exchange` (main service, receives requests, fans out to DSPs) and `dsp` (mock backend). Entrypoint is `/ad` (gzip JSON).
- **`libs/`** — Shared Go packages: `intern`, `openrtb`, `envvarutil`, `tlsutil`. Import path: `perftest/libs/...`.
- **`tools/`** — CLI tools: `genapp` (generates `d/apps.json`), `gendspconfig` (generates `d/dsps.json` from env / `d/dsp-latencies.json`).
- **`d/`** — Generated data consumed by the exchange: `apps.json`, `dsps.json`. Often gitignored or partial; use `make init` (or `make gen-apps` / `make gen-dsp-config`) to (re)generate.
- **`load-testing/`** — k6 script `k6.js`; run with `make k6`. Targets `BASE_URL` (default `http://localhost:9999`) and `/ad`.
- **`lb/`** — Nginx config; front door at port 9999 to the exchange.
- **`o11y/`** — Observability config: VictoriaMetrics, VictoriaLogs, vmalert, Grafana, Pyroscope, etc., wired via `docker-compose.yml`.

## Running things

- **Full stack:** `make up` (docker compose), then e.g. `make k6` for load test. `make down` to stop.
- **Exchange only (no Docker):** `make run-exchange` (needs `d/apps.json` and `d/dsps.json`).
- **DSP only:** `make run-dsp`.
- **CI:** `make testci`, `make lint` (see `.github/workflows/ci.yml`). Go version from `go.mod`.

## Conventions

- Go module is **`perftest`**; all internal imports use that path.
- Prefer the **Makefile** for build, run, and test commands.
- The exchange is built to be observed (Prometheus metrics, profiling); avoid removing instrumentation when changing behavior.
- When adding or changing config, consider **`.env`** and **`.env.example`**; `d/dsp-latencies.json` drives DSP latency in generated `dsps.json`.
