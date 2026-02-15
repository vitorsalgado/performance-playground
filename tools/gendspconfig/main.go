// gendspconfig generates d/dsps.json and d/dsp-latencies.json from DSP_COUNT (.env or --count).
// Latencies cycle: 0, 5ms, 10ms, 1s, 500ms.
// Usage: gendspconfig [--count <N>] [--out-dsps <path>] [--out-latencies <path>] [--env <path>]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultCount    = 25
	projectName     = "adtech"
	dspService      = "dsp"
	dspPort         = 8080
	bidPath         = "/bid"
	latencyCycleLen = 5
)

var latencyCycle = [latencyCycleLen]string{"0", "5ms", "10ms", "1s", "500ms"}

func loadEnv(path string) map[string]string {
	out := make(map[string]string)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return out
		}
		fmt.Fprintf(os.Stderr, "read .env: %v\n", err)
		os.Exit(1)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx == -1 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if key != "" {
			out[key] = val
		}
	}
	return out
}

func usage() {
	fmt.Fprintf(os.Stderr, `
Usage:
  gendspconfig [--count <N>] [--out-dsps <path>] [--out-latencies <path>] [--env <path>]

Options:
  --count         Number of DSPs (default: from .env DSP_COUNT or %d)
  --out-dsps      Output path for dsps.json (default: d/dsps.json)
  --out-latencies Output path for dsp-latencies.json (default: d/dsp-latencies.json)
  --env           Path to .env file (default: .env in cwd)
  --help          Show this help

Examples:
  gendspconfig
  gendspconfig --count 10 --out-dsps d/dsps.json --out-latencies d/dsp-latencies.json
`, defaultCount)
}

type DSPEntry struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
	Latency  string `json:"latency"`
}

func main() {
	cwd, _ := os.Getwd()
	count := flag.Int("count", -1, "number of DSPs")
	outDsps := flag.String("out-dsps", filepath.Join(cwd, "d", "dsps.json"), "output path for dsps.json")
	outLatencies := flag.String("out-latencies", filepath.Join(cwd, "d", "dsp-latencies.json"), "output path for dsp-latencies.json")
	envPath := flag.String("env", filepath.Join(cwd, ".env"), "path to .env")
	flag.Usage = usage
	flag.Parse()

	n := *count
	if n < 0 {
		env := loadEnv(*envPath)
		if v := env["DSP_COUNT"]; v != "" {
			parsed, err := strconv.Atoi(v)
			if err == nil {
				n = parsed
			}
		}
		if n < 0 {
			n = defaultCount
		}
	}
	if n < 1 {
		fmt.Fprintf(os.Stderr, "gendspconfig: invalid count: %d\n", n)
		os.Exit(1)
	}

	dsps := make([]DSPEntry, 0, n)
	latencies := make(map[string]string, n)

	for i := 1; i <= n; i++ {
		hostname := fmt.Sprintf("%s_%s_%d", projectName, dspService, i)
		latency := latencyCycle[(i-1)%latencyCycleLen]
		dsps = append(dsps, DSPEntry{
			ID:       1000 + i,
			Name:     fmt.Sprintf("dsp%d", i),
			Endpoint: fmt.Sprintf("https://%s:%d%s", hostname, dspPort, bidPath),
			Latency:  latency,
		})
		latencies[hostname] = latency
	}

	dspsDir := filepath.Dir(*outDsps)
	latenciesDir := filepath.Dir(*outLatencies)
	for _, dir := range []string{dspsDir, latenciesDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
			os.Exit(1)
		}
	}

	dspsJSON, _ := json.MarshalIndent(dsps, "", "  ")
	latenciesJSON, _ := json.MarshalIndent(latencies, "", "  ")

	if err := os.WriteFile(*outDsps, append(dspsJSON, '\n'), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write dsps: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outLatencies, append(latenciesJSON, '\n'), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write latencies: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "gendspconfig: wrote %d DSPs to %s and %s\n", n, *outDsps, *outLatencies)
}
