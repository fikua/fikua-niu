// Package ideas implements the "idees d'activitats amb previsualització de
// link" domain: a simple save/delete collection of links, each optionally
// enriched with Open Graph metadata fetched server-side (design.md ADR-01).
// It is deliberately independent from internal/items and internal/projects
// — no shared model, no NormalizeName reuse (EC-06, no deduplication in
// this space).
//
// The dependency is one-way, and only in the other direction: since
// NIU-11, internal/projects imports this package for WorkerPool and
// ValidateURL, reusing the preview machinery proven here rather than
// duplicating it. This package still imports neither internal/projects
// nor internal/items — nothing here knows those spaces exist.
//
// It deliberately imports neither net/http nor database/sql (design.md
// §4) — internal/store implements the interfaces declared here,
// internal/httpapi consumes ideas.Service only. The only network access
// this package ever triggers is through fetchsafe.FetchPreview
// (design.md ADR-02) — it never calls net/http directly.
package ideas

import (
	"context"
	"time"

	"niu/internal/items"
)

// PreviewStatus is one of the four values a row's preview_status column
// can hold (design.md §6.2). Every row is born "pending" (ADR-03) and
// moves to exactly one of the other three once the background scrape
// resolves — never back to "pending".
type PreviewStatus string

const (
	PreviewPending PreviewStatus = "pending"
	PreviewReady   PreviewStatus = "ready"
	PreviewPartial PreviewStatus = "partial"
	PreviewFailed  PreviewStatus = "failed"
)

// User is a lightweight reference to a user, embedded in Idea responses
// (added_by). Reuses items.User's shape (design.md §4) — no duplicated
// type.
type User = items.User

// Idea is the domain representation of a single row in this space.
type Idea struct {
	ID            int64
	URL           string
	Title         *string
	ImageURL      *string
	Description   *string
	PreviewStatus PreviewStatus
	AddedBy       *User
	CreatedAt     time.Time
}

// Repository is implemented by internal/store. It never leaks
// database/sql types into the domain.
type Repository interface {
	// Create inserts a new idea with preview_status='pending' and every
	// preview field NULL (ADR-03 — every row is born pending; the row is
	// returned immediately, before any scraping happens).
	Create(ctx context.Context, userID int64, url string) (Idea, error)
	// Get returns a single idea by ID. Returns ErrNotFound if absent.
	// Not exposed by Service — the only caller today is
	// IdeasRepository.Create's internal re-fetch of the row it just
	// inserted (see internal/store/ideas.go), so ErrNotFound never
	// reaches an HTTP handler in current usage (F-02, review.md).
	Get(ctx context.Context, id int64) (Idea, error)
	// List returns all ideas — a single query with a join to users for
	// added_by (no N+1).
	List(ctx context.Context) ([]Idea, error)
	// UpdatePreview is the ONLY UPDATE allowed once a row has been
	// created (design.md §4) — it never touches id/url/added_by. It
	// writes the scrape result (title/image_url/description, each
	// possibly nil) and the final preview_status ("ready"/"partial"/
	// "failed"). If id no longer exists (the idea was deleted while the
	// scrape was in flight, design.md §5 Flux 3), it affects zero rows
	// and returns no error — the caller (the worker pool) is not
	// expected to treat that as a failure.
	UpdatePreview(ctx context.Context, id int64, title, imageURL, description *string, status PreviewStatus) error
	// Delete removes the row. Implementations MUST be idempotent:
	// deleting an already-absent id is not an error (EC-15).
	Delete(ctx context.Context, id int64) (existed bool, err error)
}

// EventSink records domain events to the append-only events table
// (PLAN.md §2.4) — the same mechanism items.EventSink/projects.EventSink
// already use, no new column or table.
type EventSink interface {
	// Record writes one event. kind is one of "idea_added",
	// "idea_preview_resolved", "idea_deleted"; payload is a
	// JSON-serialisable value.
	Record(ctx context.Context, userID int64, kind string, payload any) error
}
