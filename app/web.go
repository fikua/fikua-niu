package niu

import "embed"

// WebFS embeds the static frontend (design.md §8, T-15). Declared at the
// module root for the same reason as MigrationsFS — //go:embed cannot
// reach outside its own directory subtree.
//
//go:embed all:web
var WebFS embed.FS
