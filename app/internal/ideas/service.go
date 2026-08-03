package ideas

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
)

// PreviewFetcher is the single seam through which Service reaches
// fetchsafe.FetchPreview — declared here (rather than importing
// internal/fetchsafe's *http.Client type directly) so this package still
// does not import net/http (design.md §4). main.go wires the real
// fetchsafe.FetchPreview (bound to its dedicated client, T-03h) as this
// function value.
type PreviewFetcher func(ctx context.Context, rawURL string) (Preview, error)

// Preview mirrors fetchsafe.Preview's shape without importing
// internal/fetchsafe's package directly into this file's exported
// surface — kept structurally identical on purpose so main.go's wiring is
// a straight function value, no adapter needed.
type Preview struct {
	Title       string
	ImageURL    string
	Description string
	Partial     bool
}

// Service implements the "idees d'activitats" business rules. It depends
// only on the Repository and EventSink interfaces, plus a PreviewFetcher
// function value — never on database/sql or net/http directly (design.md
// §4). The only network access ever triggered by this package flows
// through fetch, which callers wire to fetchsafe.FetchPreview.
type Service struct {
	repo  Repository
	sink  EventSink
	fetch PreviewFetcher
	pool  *WorkerPool
}

// NewService constructs a Service. pool must already be started (see
// NewWorkerPool) — Service.Add submits scrape jobs to it, never spawns an
// unbounded goroutine per request (ADR-03, F-05).
func NewService(repo Repository, sink EventSink, fetch PreviewFetcher, pool *WorkerPool) *Service {
	return &Service{repo: repo, sink: sink, fetch: fetch, pool: pool}
}

// validateURL trims and validates the raw URL: rejects empty/whitespace-
// only input (EC-10) and validates ONLY the scheme (http/https,
// NFR-05/EC-01) — the same cheap, network-free check fetchsafe itself
// performs first, reused here so Service.Add never creates a row for an
// input that fetchsafe would immediately reject anyway. No network
// request happens during this validation.
func validateURL(rawURL string) (string, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", ErrValidation{
			Code:    ValidationEmpty,
			Message: "Escriu un enllaç abans d'afegir.",
		}
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", ErrValidation{
			Code:    ValidationInvalidFormat,
			Message: "Aquest enllaç no és vàlid.",
		}
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" {
		return "", ErrValidation{
			Code:    ValidationSchemeRejected,
			Message: "Aquest enllaç no és vàlid — ha de començar per http:// o https://.",
		}
	}

	return trimmed, nil
}

// Add validates the URL, inserts the row immediately with
// preview_status='pending' (design.md §5 Flux 1, ADR-03 — the 201
// response never waits for the scrape), and enqueues the scrape onto the
// bounded worker pool. Covers AC-01 (base), AC-02 (base), EC-01, EC-10.
func (s *Service) Add(ctx context.Context, userID int64, rawURL string) (Idea, error) {
	validURL, err := validateURL(rawURL)
	if err != nil {
		return Idea{}, err
	}

	idea, err := s.repo.Create(ctx, userID, validURL)
	if err != nil {
		return Idea{}, err
	}

	_ = s.sink.Record(ctx, userID, "idea_added", map[string]any{
		"idea_id": idea.ID,
		"url":     idea.URL,
	})

	s.pool.Submit(idea.ID, idea.URL, s.resolvePreview)

	return idea, nil
}

// resolvePreview is the work a pool worker performs for one idea: call
// fetchsafe.FetchPreview (via the injected fetch function, with the
// worker pool's own background context — never the original request's
// context, which is already gone by the time this runs, ADR-03), then
// persist the outcome via UpdatePreview and record idea_preview_resolved.
//
// Any error from fetch — forbidden destination, timeout, oversized
// response, unsupported content type, or no recognizable OG tags — is
// treated identically: preview_status='failed', every preview field left
// NULL. This is a deliberate, permanent fallback (never retried
// automatically, requirements.md §7) and it is indistinguishable to the
// end user from any other fallback reason (NFR-06 — no error detail is
// ever surfaced for "destination forbidden" specifically).
func (s *Service) resolvePreview(ctx context.Context, ideaID int64, rawURL string) {
	preview, err := s.fetch(ctx, rawURL)

	var title, imageURL, description *string
	status := PreviewFailed

	if err == nil {
		status = PreviewReady
		if preview.Partial {
			status = PreviewPartial
		}
		if preview.Title != "" {
			title = &preview.Title
		}
		if preview.ImageURL != "" {
			imageURL = &preview.ImageURL
		}
		if preview.Description != "" {
			description = &preview.Description
		}
		// A "partial" result with literally nothing recovered is
		// indistinguishable from a full fallback from the user's point of
		// view (proposal.md §8.2 Estat B vs C) — treat it as failed so the
		// card renders the fallback state, not an empty "complete" card.
		if title == nil && imageURL == nil && description == nil {
			status = PreviewFailed
		}
	} else {
		slog.Debug("fetchsafe: preview resolution failed", "idea_id", ideaID, "error", err)
	}

	if updateErr := s.repo.UpdatePreview(ctx, ideaID, title, imageURL, description, status); updateErr != nil {
		// The idea may have been deleted while the scrape was in flight
		// (design.md §5 Flux 3) — UpdatePreview affecting zero rows is not
		// surfaced as an error by the repository, so any error reaching
		// here is a genuine unexpected failure worth logging.
		slog.Error("ideas: failed to persist preview resolution", "idea_id", ideaID, "error", updateErr)
		return
	}

	_ = s.sink.Record(ctx, 0, "idea_preview_resolved", map[string]any{
		"idea_id": ideaID,
		"status":  string(status),
	})
}

// Delete removes an idea, idempotently (EC-15): a second call on an
// already-deleted id also succeeds without error and without writing a
// second event. If a scrape for this id is still in flight when this
// runs, its later UpdatePreview simply affects zero rows and is ignored
// (design.md §5 Flux 3) — no explicit cancellation.
func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	existed, err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	if existed {
		_ = s.sink.Record(ctx, userID, "idea_deleted", map[string]any{
			"idea_id": id,
		})
	}
	return nil
}

// List returns every idea — a single query, no N+1 (AC-04 base, NFR-09
// base: never triggers any scrape, only reads persisted state).
func (s *Service) List(ctx context.Context) ([]Idea, error) {
	return s.repo.List(ctx)
}
