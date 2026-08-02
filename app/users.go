package niu

import "embed"

// UsersConfigFS embeds users.json — the non-secret identity fields
// (name, display name, avatar emoji) for the two seeded users. Declared
// at the module root for the same reason as WebFS and MigrationsFS:
// //go:embed cannot reach outside its own directory subtree.
//
// This file is deliberately committed and versioned. Only credentials
// that are actually secret (NIU_SESSION_SECRET, the bcrypt password
// hashes) stay in the host-only .env — a username, a display name and an
// avatar emoji carry no security value, and forcing them through the
// same manual, error-prone .env editing process as real secrets was
// exactly what caused repeated deploy friction (2026-08-02): a
// forgotten NIU_USER_A_AVATAR silently fell back to the default, and a
// display-name typo required SSHing into the VPS to fix.
//
//go:embed users.json
var UsersConfigFS embed.FS
