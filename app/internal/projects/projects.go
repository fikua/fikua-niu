// Package projects implements the "compres grans i projectes de casa"
// domain: a three-state lifecycle (idea -> decidit -> fet, reversible in
// both directions) independent from internal/items (design.md ADR-01). It
// deliberately imports neither net/http nor database/sql (design.md §4) —
// internal/store implements the interfaces declared here, internal/httpapi
// consumes projects.Service only. It imports internal/items only for
// items.NormalizeName (ADR-02) — no other symbol.
//
// NIU-11 adds an optional link preview (url/title/image_url/description/
// preview_status), replicating the pattern already proved in
// internal/ideas (NIU-6) rather than inventing a new one — same
// PreviewFetcher seam, same WorkerPool, same four-value preview_status.
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

// Preview status values a project's PreviewStatus column can hold once a
// url was given (tasks.md T-01). Declared as plain string constants, not
// a typed enum like ideas.PreviewStatus, because the field itself is
// *string here (NULL-able, T-02) — a project without a url never has one
// of these values, not even PreviewPending: PreviewStatus stays nil.
const (
	PreviewPending = "pending"
	PreviewReady   = "ready"
	PreviewPartial = "partial"
	PreviewFailed  = "failed"
)

// User is a lightweight reference to a user, embedded in Project responses
// (added_by / last_updated_by). Reuses items.User's shape (design.md §4) —
// no duplicated type, just an alias so this package's public surface makes
// the reuse explicit.
type User = items.User

// Project is the domain representation of a single row in this space.
//
// URL/Title/ImageURL/Description/PreviewStatus are all optional (NIU-11):
// a project created without a url leaves every one of them nil, including
// PreviewStatus itself — unlike ideas.Idea, where PreviewStatus is never
// nil (every idea always has a url and is therefore always born
// "pending"). Here nil PreviewStatus means "no preview was ever
// requested", not "not yet resolved".
type Project struct {
	ID            int64
	Name          string
	State         State
	Budget        *string
	TargetDate    *string
	URL           *string
	Title         *string
	ImageURL      *string
	Description   *string
	PreviewStatus *string
	AddedBy       *User
	LastUpdatedBy *User
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// PreviewFetcher is the single seam through which Service reaches
// fetchsafe.FetchPreview — declared here (rather than importing
// internal/fetchsafe's *http.Client type directly) so this package still
// does not import net/http (design.md §4). main.go wires the real
// fetchsafe.FetchPreview (bound to its dedicated client, T-03h of NIU-6)
// as this function value — the exact same client internal/ideas already
// uses, never a second one (tasks.md T-06).
type PreviewFetcher func(ctx context.Context, rawURL string) (Preview, error)

// Preview mirrors fetchsafe.Preview's shape without importing
// internal/fetchsafe's package directly into this file's exported
// surface — kept structurally identical to ideas.Preview on purpose, so
// main.go's wiring is a straight function value, no adapter needed.
type Preview struct {
	Title       string
	ImageURL    string
	Description string
	Partial     bool
}

// Repository is implemented by internal/store. It never leaks
// database/sql types into the domain.
type Repository interface {
	// Create inserts a new project and returns it with its assigned ID.
	// Implementations MUST check ExistsByNormalizedName inside the same
	// transaction as the INSERT (ADR-02 — closes the check-then-insert
	// race via the DB-level unique index), across ALL states (EC-03). url
	// is optional (nil when the project was created without one, tasks.md
	// T-01/T-04) — when non-nil the row is inserted with
	// preview_status='pending', mirroring ideas.Repository.Create; when
	// nil, preview_status stays NULL (never 'pending' — there is no
	// preview to resolve).
	Create(ctx context.Context, userID int64, name, nameNormalized string, budget, targetDate, url *string) (Project, error)
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
	// UpdatePreview is the ONLY UPDATE allowed for the preview fields
	// (mirrors ideas.Repository.UpdatePreview, tasks.md T-03) — it never
	// touches id/name/state/budget/target_date/url. It writes the scrape
	// result (title/image_url/description, each possibly nil) and the
	// final preview_status ("ready"/"partial"/"failed"). If id no longer
	// exists (the project was deleted while the scrape was in flight), it
	// affects zero rows and returns no error — zero rows affected is not
	// an error here, it is the expected outcome of that race.
	UpdatePreview(ctx context.Context, id int64, title, imageURL, description *string, status string) error
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
