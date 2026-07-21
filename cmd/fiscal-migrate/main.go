// Command fiscal-migrate applies embedded forward-only schema migrations (cloud deploy).
// Production CLI exposes only: up, version.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/db"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/platform/dbmigrate"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) < 1 {
		printUsage(os.Stderr)
		return 2
	}
	cmd := args[0]
	switch cmd {
	case "up", "version":
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q (only up and version are supported)\n", cmd)
		printUsage(os.Stderr)
		return 2
	}

	driver := strings.TrimSpace(os.Getenv("FISCAL_DATABASE_DRIVER"))
	dsn := strings.TrimSpace(os.Getenv("FISCAL_DATABASE_URL"))
	if driver == "" || dsn == "" {
		fmt.Fprintln(os.Stderr, "FISCAL_DATABASE_DRIVER and FISCAL_DATABASE_URL are required")
		return 2
	}

	dialect, err := dialectFromDriver(driver)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 2
	}

	switch cmd {
	case "up":
		if err := dbmigrate.Up(dialect, dsn); err != nil {
			fmt.Fprintf(os.Stderr, "migrate up: %v\n", err)
			return 1
		}
		v, dirty, err := dbmigrate.Version(dialect, dsn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "migrate version: %v\n", err)
			return 1
		}
		fmt.Printf("ok version=%d dirty=%v\n", v, dirty)
		return 0
	case "version":
		v, dirty, err := dbmigrate.Version(dialect, dsn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "migrate version: %v\n", err)
			return 1
		}
		fmt.Printf("version=%d dirty=%v\n", v, dirty)
		return 0
	default:
		return 2
	}
}

func dialectFromDriver(driver string) (dbmigrate.Dialect, error) {
	switch driver {
	case db.DriverPostgres, "pgx", "postgresql":
		return dbmigrate.DialectPostgres, nil
	case db.DriverSQLite:
		return dbmigrate.DialectSQLite, nil
	default:
		return "", fmt.Errorf("unsupported FISCAL_DATABASE_DRIVER %q", driver)
	}
}

func printUsage(w *os.File) {
	fmt.Fprintln(w, "usage: fiscal-migrate <up|version>")
	fmt.Fprintln(w, "env: FISCAL_DATABASE_DRIVER=postgres|sqlite")
	fmt.Fprintln(w, "     FISCAL_DATABASE_URL=<dsn or path>")
}
