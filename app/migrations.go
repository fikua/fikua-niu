package niu

import "embed"

// MigrationsFS embeds the goose SQL migrations (design.md §8, T-01/T-15).
// Declared at the module root because Go's //go:embed can only reach
// files within the same directory subtree as the source file — cmd/niu
// imports this package to pass the embedded FS into store.Open.
//
//go:embed all:migrations
var MigrationsFS embed.FS
