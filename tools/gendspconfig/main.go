// gendspconfig generates d/dsps.json from DSP_COUNT (.env or --count).
// Each DSP entry's latency is read from d/dsp-latencies.json (array by index); missing index → "0".
// Usage: gendspconfig [--count <N>] [--out-dsps <path>] [--latencies <path>] [--env <path>]
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
	defaultCount = 25
	projectName  = "adtech"
	dspService   = "dsp"
	dspPort      = 8080
	bidPath      = "/bid"
	defaultLatency = "0"
)

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
  gendspconfig [--count <N>] [--out-dsps <path>] [--latencies <path>] [--env <path>]

Options:
  --count      Number of DSPs (default: from .env DSP_COUNT or %d)
  --out-dsps   Output path for dsps.json (default: d/dsps.json)
  --latencies  Path to dsp-latencies.json array (default: d/dsp-latencies.json); index = DSP index 1..n, missing → "0"
  --env        Path to .env file (default: .env in cwd)
  --help       Show this help

Examples:
  gendspconfig
  gendspconfig --count 10 --out-dsps d/dsps.json --latencies d/dsp-latencies.json
`, defaultCount)
}

// loadLatencies reads a JSON array of latency strings from path. Missing or invalid file returns nil (all "0").
func loadLatencies(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "gendspconfig: read latencies: %v\n", err)
		}
		return nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		fmt.Fprintf(os.Stderr, "gendspconfig: parse latencies: %v\n", err)
		return nil
	}
	return arr
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
	latenciesPath := flag.String("latencies", filepath.Join(cwd, "d", "dsp-latencies.json"), "path to dsp-latencies.json array")
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

	latencies := loadLatencies(*latenciesPath)

	dsps := make([]DSPEntry, 0, n)

	for i := 1; i <= n; i++ {
		// Match Docker Compose v2 container names (project-service-replica) for DNS resolution.
		hostname := fmt.Sprintf("%s-%s-%d", projectName, dspService, i)
		latency := defaultLatency
		if idx := i - 1; idx < len(latencies) && latencies[idx] != "" {
			latency = latencies[idx]
		}
		dsps = append(dsps, DSPEntry{
			ID:       1000 + i,
			Name:     fmt.Sprintf("dsp%d", i),
			Endpoint: fmt.Sprintf("https://%s:%d%s", hostname, dspPort, bidPath),
			Latency:  latency,
		})
	}

	dspsDir := filepath.Dir(*outDsps)
	if err := os.MkdirAll(dspsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	dspsJSON, _ := json.MarshalIndent(dsps, "", "  ")

	if err := os.WriteFile(*outDsps, append(dspsJSON, '\n'), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write dsps: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "gendspconfig: wrote %d DSPs to %s\n", n, *outDsps)
}
