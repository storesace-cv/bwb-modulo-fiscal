// Command fiscal-sandbox-measure runs closed S3C1 measurement profiles.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/sandboxmeasure"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	profile, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: measure_failed\n")
		return 1
	}
	fixture, err := sandboxmeasure.DefaultFixturePath(os.Args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: measure_failed\n")
		return 1
	}
	cfg := sandboxmeasure.Config{
		Profile:           profile,
		BaseURL:           sandboxmeasure.FixedBaseURL,
		TokenPath:         sandboxmeasure.FixedTokenFile,
		FixturePath:       fixture,
		EnforceAcceptance: true,
	}
	rep, err := sandboxmeasure.Run(context.Background(), cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: measure_failed\n")
		return 1
	}
	if err := sandboxmeasure.WriteReport(os.Stdout, rep); err != nil {
		fmt.Fprintf(os.Stderr, "error: measure_failed\n")
		return 1
	}
	return 0
}

func parseArgs(args []string) (sandboxmeasure.Profile, error) {
	if len(args) != 2 || args[0] != "--profile" {
		return "", fmt.Errorf("usage")
	}
	return sandboxmeasure.ParseProfile(args[1])
}
