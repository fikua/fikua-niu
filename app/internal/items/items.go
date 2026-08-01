// Package items implements the shopping-list/pantry domain: item
// validation, duplicate-name normalisation (ADR-02) and location
// transitions. It deliberately imports neither net/http nor database/sql
// (design.md §4) — internal/store implements the interfaces declared
// here, internal/httpapi consumes items.Service only.
package items

import (
	"context"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"
)

// Location is one of the two boxes an item can live in.
type Location string

const (
	LocationShopping Location = "shopping"
	LocationPantry   Location = "pantry"
)

// User is a lightweight reference to a user, embedded in Item responses
// (added_by / moved_by).
type User struct {
	ID          int64
	Name        string
	DisplayName string
	AvatarEmoji string
}

// Item is the domain representation of a single shopping-list/pantry row.
type Item struct {
	ID        int64
	Name      string
	Location  Location
	Position  float64
	AddedBy   *User
	MovedBy   *User
	MovedAt   *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Repository is implemented by internal/store. It never leaks
// database/sql types into the domain.
type Repository interface {
	// Create inserts a new item and returns it with its assigned ID.
	// Implementations MUST check ExistsByNormalizedName inside the same
	// transaction as the INSERT (ADR-02 — closes the check-then-insert
	// race via the DB-level unique index).
	Create(ctx context.Context, userID int64, name, nameNormalized string) (Item, error)
	// Get returns a single item by ID. Returns ErrNotFound if absent.
	Get(ctx context.Context, id int64) (Item, error)
	// List returns all items ordered by location, position — a single
	// query with joins to users for added_by/moved_by (no N+1, NFR-05).
	List(ctx context.Context) ([]Item, error)
	// Update applies a location move: sets location, moved_by, moved_at,
	// updated_at and position, all inside a single transaction (ADR-01).
	// Returns ErrNotFound if the id does not exist.
	Update(ctx context.Context, id, userID int64, newLocation Location, position float64) (Item, error)
	// Delete removes the row. Implementations MUST be idempotent: deleting
	// an already-absent id is not an error (EC-11).
	Delete(ctx context.Context, id int64) (existed bool, err error)
	// ExistsByNormalizedName reports whether an active item with this
	// normalized name already exists, and if so, in which location.
	ExistsByNormalizedName(ctx context.Context, nameNormalized string) (exists bool, location Location, err error)
	// MaxPosition returns the highest position value currently used in
	// the given location, and whether the location has any items at all.
	MaxPosition(ctx context.Context, location Location) (max float64, hasItems bool, err error)
}

// EventSink records domain events to the append-only events table
// (PLAN.md §2.4) — substrate for future gamification, unused for reads in
// NIU-1.
type EventSink interface {
	// Record writes one event. kind is one of "item_added", "item_moved",
	// "item_deleted"; payload is a JSON-serialisable value.
	Record(ctx context.Context, userID int64, kind string, payload any) error
}

// NormalizeName implements ADR-02's exact three-step algorithm, in this
// exact order:
//
//  1. norm.NFC.String(...)   — Unicode canonical composition. NOT
//     optional: without it, the same visible text arriving as NFC
//     (typical keyboards/browsers) vs NFD (historically macOS) compares
//     unequal under ToLower alone, letting a duplicate slip past EC-06.
//  2. strings.TrimSpace(...) — trims leading/trailing whitespace.
//  3. strings.ToLower(...)   — Unicode-aware case folding.
//
// items.name (the value shown to the user) is NEVER normalized — only
// name_normalized is derived via this function, for uniqueness checks.
func NormalizeName(raw string) string {
	composed := norm.NFC.String(raw)
	trimmed := strings.TrimSpace(composed)
	return strings.ToLower(trimmed)
}
