// genapp generates a JSON array of App records for the exchange (d/apps.json).
// Usage: genapp --count <N> [--out <path|-] [--publisher-count <N>] [--start-id <N>]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func usage() {
	fmt.Fprint(os.Stderr, `
Usage:
  genapp --count <N> [--out <path|-] [--publisher-count <N>] [--start-id <N>]

Options:
  --count             Number of App records to generate (required, integer >= 0)
  --out               Output file path, or "-" for stdout (default: "-")
  --publisher-count   Number of distinct publishers to rotate through (default: 1000)
  --start-id          Starting App id (default: 1)
  --help              Show this help

Examples:
  genapp --count 1000 --out d/apps.json
  genapp --count 500000 --publisher-count 500 --start-id 1250 --out d/apps.json
`)
}

type Publisher struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type App struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	Publisher *Publisher `json:"publisher"`
}

func makeApp(appID, publisherCount int) *App {
	pubID := ((appID - 1) % publisherCount) + 1
	return &App{
		ID:   appID,
		Name: fmt.Sprintf("app-%d", appID),
		Publisher: &Publisher{
			ID:   pubID,
			Name: fmt.Sprintf("publisher-%d", pubID),
		},
	}
}

func main() {
	count := flag.Int("count", -1, "number of App records")
	outPath := flag.String("out", "-", "output path or - for stdout")
	publisherCount := flag.Int("publisher-count", 1000, "distinct publishers to rotate")
	startID := flag.Int("start-id", 1, "starting App id")
	flag.Usage = usage
	flag.Parse()

	if *count < 0 {
		fmt.Fprintln(os.Stderr, "missing required --count (must be >= 0)")
		usage()
		os.Exit(2)
	}
	if *publisherCount <= 0 {
		fmt.Fprintln(os.Stderr, "--publisher-count must be > 0")
		os.Exit(2)
	}
	if *startID < 0 {
		fmt.Fprintln(os.Stderr, "--start-id must be >= 0")
		os.Exit(2)
	}

	var out *os.File
	if *outPath == "-" {
		out = os.Stdout
	} else {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create output: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		out = f
	}

	if _, err := out.Write([]byte("[")); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}

	for i := 0; i < *count; i++ {
		appID := *startID + i
		app := makeApp(appID, *publisherCount)
		if i > 0 {
			if _, err := out.Write([]byte(",")); err != nil {
				fmt.Fprintf(os.Stderr, "write: %v\n", err)
				os.Exit(1)
			}
		}
		b, err := json.Marshal(app)
		if err != nil {
			fmt.Fprintf(os.Stderr, "encode: %v\n", err)
			os.Exit(1)
		}
		if _, err := out.Write(b); err != nil {
			fmt.Fprintf(os.Stderr, "write: %v\n", err)
			os.Exit(1)
		}
	}

	if _, err := out.Write([]byte("]")); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}

	if out != os.Stdout {
		fmt.Fprintf(os.Stderr, "genapp: wrote %d apps to %s\n", *count, *outPath)
	}
}
