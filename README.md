# Performance Playground · [![CI](https://github.com/vitorsalgado/ad-tech-performance/actions/workflows/ci.yml/badge.svg)](https://github.com/vitorsalgado/ad-tech-performance/actions/workflows/ci.yml) · ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/vitorsalgado/performance-playground) · ![GitHub License](https://img.shields.io/github/license/vitorsalgado/performance-playground)

This project offers a "playground" where users can run performance experiments.  
It implements a very simple ad exchange like system with a complete observability stack already set.  
The experiment application was intentionally developed with some known inefficiencies, so we can play around, experiment, and see the actual impact in the system metrics.

## Getting Started

The whole system is deployed with the [docker-compose.yml](docker-compose.yml) at the project root.  
To run everything, execute the following command:

```
make up
```

To stop everything, execute:

```
make down
```

Check the [Makefile](Makefile) for all the commands available.

The [docker-compose.yml](docker-compose.yml) will deploy everything inter-connected and ready to be used.

## Testing

The project comes with a basic load testing script based on [k6](https://k6.io/).  
Check it out at [load-testing](load-testing/k6.js).  
To run the load test, execute:

```
make k6
```

### cURL

A simple cURL to run an HTTP request against the ad exchange, through the load balancer:

```
curl -sS -X POST 'http://localhost:9999/ad' -H 'Content-Type: application/json' -H 'Content-Encoding: gzip' --data-binary @<(printf '%s' '{"id":"1","imp":[{"id":"1","banner":{"w":300,"h":250}}],"app":{"id":"1250","publisher":{"id":"1"}}}' | gzip -c)
```

### Tools

The project contains scripts to generate data for the ad exchange experiment application. Check the scripts in the [tools](tools) directory.  
The data will be saved at the directory [d](d), at the project root.

## Tech

The very basic ad exchange is implemented with **Go**. You can find it at [flavors/adtech](flavors/adtech/).  
The entrypoint for the system is a Load Balancer implemented with Nginx. Check it out at [lb](lb).  
Observability is heavily based on [VictoriaMetrics](https://victoriametrics.com/) stack, with Grafana being used for metrics visualization.  
For continuous-profiling, [Pyroscope](https://grafana.com/docs/pyroscope/latest/) is used.  
The observability stack is configured in the directory [o11y](o11y), and deployed using this [docker-compose](docker-compose.yml).  

### Stack

Here is the list of the whole stack:

- Go (basic ad exchange).
- Nginx (Load Balancer).
- VictoriaMetrics.
- VictoriaLogs.
- VM Alerts.
- Alertmanager.
- Grafana.
- Pyroscope.

## License

This project is [MIT licensed](LICENSE).
