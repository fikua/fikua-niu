// Package projects implements the "compres grans i projectes de casa"
// domain: a three-state lifecycle (idea -> decidit -> fet, reversible in
// both directions) independent from internal/items (design.md ADR-01). It
// deliberately imports neither net/http nor database/sql (design.md §4) —
// internal/store implements the interfaces declared here, internal/httpapi
// consumes projects.Service only. It imports internal/items only for
// items.NormalizeName (ADR-02) — no other symbol.
package projects

import (
	"context"
	"time"

	"niu/internal/items"
)

// State is one of the three stages a project can be in. All three are
// always a valid move from any other (AC-09) — there is no forbidden
// transition to enforce.
type State string

const (
	StateIdea    State = "idea"
	StateDecidit State = "decidit"
	StateFet     State = "fet"
)

// User is a lightweight reference to a user, embedded in Project responses
// (added_by / last_updated_by). Reuses items.User's shape (design.md §4) —
// no duplicated type, just an alias so this package's public surface makes
// the reuse explicit.
type User = items.User

// Project is the domain representation of a single row in this space.
type Project struct {
	ID            int64
	Name          string
	State         State
	Budget        *string
	TargetDate    *string
	AddedBy       *User
	LastUpdatedBy *User
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Repository is implemented by internal/store. It never leaks
// database/sql types into the domain.
type Repository interface {
	// Create inserts a new project and returns it with its assigned ID.
	// Implementations MUST check ExistsByNormalizedName inside the same
	// transaction as the INSERT (ADR-02 — closes the check-then-insert
	// race via the DB-level unique index), across ALL states (EC-03).
	Create(ctx context.Context, userID int64, name, nameNormalized string, budget, targetDate *string) (Project, error)
	// Get returns a single project by ID. Returns ErrNotFound if absent.
	Get(ctx context.Context, id int64) (Project, error)
	// List returns all projects — a single query with joins to users for
	// added_by/last_updated_by (no N+1).
	List(ctx context.Context) ([]Project, error)
	// UpdateState applies a state transition: sets state, last_updated_by,
	// updated_at, inside a single BEGIN IMMEDIATE transaction (ADR-01 of
	// NIU-1, reapplied here — design.md §5 Flux 2). The prior state is read
	// inside that same transaction, immediately before the UPDATE, and
	// returned as previousState — this is the only way to guarantee the
	// value is what this specific commit truly overwrote, not a value read
	// by a separate, non-transactional round-trip that a concurrent writer
	// could invalidate in between (F-23). Returns ErrNotFound if the id
	// does not exist.
	UpdateState(ctx context.Context, id, userID int64, newState State) (project Project, previousState State, err error)
	// Delete removes the row. Implementations MUST be idempotent: deleting
	// an already-absent id is not an error (EC-13).
	Delete(ctx context.Context, id int64) (existed bool, err error)
	// ExistsByNormalizedName reports whether a project with this
	// normalized name already exists, in ANY state (EC-03 — wider scope
	// than items.Repository.ExistsByNormalizedName, which only checks
	// active items).
	ExistsByNormalizedName(ctx context.Context, nameNormalized string) (bool, error)
}

// EventSink records domain events to the append-only events table
// (PLAN.md §2.4) — the same mechanism items.EventSink already uses, no new
// column or table.
type EventSink interface {
	// Record writes one event. kind is one of "project_added",
	// "project_state_changed", "project_deleted"; payload is a
	// JSON-serialisable value.
	Record(ctx context.Context, userID int64, kind string, payload any) error
}

// NormalizeName reuses items.NormalizeName verbatim (ADR-02) — the same
// trim + NFC + lowercase algorithm, never duplicated.
func NormalizeName(raw string) string {
	return items.NormalizeName(raw)
}
