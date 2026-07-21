// Package migrations embeds forward-only SQL migration files for golang-migrate.
package migrations

import "embed"

//go:embed postgres/*.sql sqlite/*.sql
var FS embed.FS
