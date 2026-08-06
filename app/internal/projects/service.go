package projects

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"niu/internal/ideas"
)

const maxNameLength = 200
const maxBudgetLength = 200

// targetDateLayout is the ISO-8601 date-only format (YYYY-MM-DD) required
// by AC-15/EC-17 — no time component, no rejection of past dates.
const targetDateLayout = "2006-01-02"

// hasControlChars reports whether s contains any character that must not
// be stored in a project name — same discipline as items.hasControlChars
// (EC-08/NFR-02, Trojan Source, zero-width bypass of duplicate rules).
func hasControlChars(s string) bool {
	for _, r := range s {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			return true
		case unicode.IsControl(r):
			return true
		case unicode.Is(unicode.Cf, r):
			return true
		}
	}
	return false
}

// Service implements the "compres grans i projectes de casa" business
// rules. It depends only on the Repository and EventSink interfaces, plus
// a PreviewFetcher function value (NIU-11) — never on database/sql or
// net/http directly (design.md §4). The only network access ever
// triggered by this package flows through fetch, which callers wire to
// fetchsafe.FetchPreview — same discipline as internal/ideas.Service.
type Service struct {
	repo  Repository
	sink  EventSink
	fetch PreviewFetcher
	pool  *ideas.WorkerPool
}

// NewService constructs a Service. pool must already be started (see
// ideas.NewWorkerPool) — Service.Add submits scrape jobs to it, never
// spawns an unbounded goroutine per request (mirrors ideas.NewService).
// tasks.md T-06 wires this to the SAME pool internal/ideas already uses
// (cmd/niu/main.go's previewPool) rather than a second one — see that
// file for the rationale.
func NewService(repo Repository, sink EventSink, fetch PreviewFetcher, pool *ideas.WorkerPool) *Service {
	return &Service{repo: repo, sink: sink, fetch: fetch, pool: pool}
}

// validateName trims and validates a project name, applying the same
// 1-200 threshold and control-character rules as internal/items (AC-10,
// EC-01, EC-02).
func validateName(rawName string) (string, error) {
	trimmed := strings.TrimSpace(rawName)

	if trimmed == "" {
		return "", ErrValidation{
			Code:    ValidationEmpty,
			Message: "Escriu un nom abans d'afegir.",
		}
	}

	if utf8.RuneCountInString(trimmed) > maxNameLength {
		return "", ErrValidation{
			Code:    ValidationTooLong,
			Message: "Massa llarg — màxim 200 caràcters.",
		}
	}

	if hasControlChars(rawName) {
		return "", ErrValidation{
			Code:    ValidationControlChars,
			Message: "Aquest nom conté caràcters no vàlids.",
		}
	}

	return trimmed, nil
}

// validateBudget trims and validates the optional budget field: nil/empty
// is allowed (AC-14), otherwise the same 1-200 threshold as the name
// applies (EC-16). Returns nil when no budget was provided.
func validateBudget(rawBudget *string) (*string, error) {
	if rawBudget == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*rawBudget)
	if trimmed == "" {
		return nil, nil
	}

	if utf8.RuneCountInString(trimmed) > maxBudgetLength {
		return nil, ErrValidation{
			Code:    ValidationBudgetTooLong,
			Message: "Pressupost massa llarg — màxim 200 caràcters.",
		}
	}

	if hasControlChars(*rawBudget) {
		return nil, ErrValidation{
			Code:    ValidationControlChars,
			Message: "Aquest pressupost conté caràcters no vàlids.",
		}
	}

	return &trimmed, nil
}

// validateTargetDate validates the optional target_date field: nil/empty
// is allowed (AC-15), otherwise it must parse as a valid ISO-8601
// YYYY-MM-DD date. EC-17 explicitly requires accepting a date in the past
// without error — no range check beyond "is this a valid calendar date".
func validateTargetDate(rawTargetDate *string) (*string, error) {
	if rawTargetDate == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*rawTargetDate)
	if trimmed == "" {
		return nil, nil
	}

	if _, err := time.Parse(targetDateLayout, trimmed); err != nil {
		return nil, ErrValidation{
			Code:    ValidationInvalidDate,
			Message: "Data objectiu no vàlida — utilitza el format AAAA-MM-DD.",
		}
	}

	return &trimmed, nil
}

// validateProjectURL validates the optional url field: nil/empty is
// allowed (decision 2, tasks.md context — a project without a url is not
// an error, unlike ideas.Service.Add where the url is mandatory).
// Non-empty input reuses ideas.ValidateURL's exact scheme check
// (tasks.md T-04: "reutilitzar la validació d'esquema — no duplicar-la a
// mà") rather than hand-duplicating the http(s)-only rule, wrapping the
// result into this package's own ErrValidation so httpapi/frontend keep
// a single error-code namespace for projects.
func validateProjectURL(rawURL *string) (*string, error) {
	if rawURL == nil {
		return nil, nil
	}

	trimmed := strings.TrimSpace(*rawURL)
	if trimmed == "" {
		return nil, nil
	}

	validURL, err := ideas.ValidateURL(trimmed)
	if err != nil {
		var val ideas.ErrValidation
		if errors.As(err, &val) {
			return nil, ErrValidation{Code: val.Code, Message: val.Message}
		}
		return nil, ErrValidation{Code: ValidationURLInvalid, Message: "Aquest enllaç no és vàlid."}
	}

	return &validURL, nil
}

// Add validates and creates a new project in state "idea" (design.md §5
// Flux 1; covers AC-01, AC-10, AC-14, AC-15, EC-01, EC-02, EC-03, EC-07,
// EC-16, EC-17). NIU-11: when a url is given, it also enqueues the
// og:image/title/description scrape onto the shared preview worker pool
// (tasks.md T-04/T-06) — the 201-equivalent response never waits for it,
// same ADR-03 rule already proved by internal/ideas. A project with no
// url enqueues nothing and is created with preview_status left NULL.
func (s *Service) Add(ctx context.Context, userID int64, rawName string, rawBudget, rawTargetDate, rawURL *string) (Project, error) {
	name, err := validateName(rawName)
	if err != nil {
		return Project{}, err
	}

	budget, err := validateBudget(rawBudget)
	if err != nil {
		return Project{}, err
	}

	targetDate, err := validateTargetDate(rawTargetDate)
	if err != nil {
		return Project{}, err
	}

	projectURL, err := validateProjectURL(rawURL)
	if err != nil {
		return Project{}, err
	}

	nameNormalized := NormalizeName(name)

	// Duplicate check + INSERT happen inside the same transaction at the
	// store layer (ADR-02), across ALL states (EC-03) — Create is
	// responsible for that atomicity and for surfacing ErrDuplicate when
	// the DB-level unique index rejects the insert.
	project, err := s.repo.Create(ctx, userID, name, nameNormalized, budget, targetDate, projectURL)
	if err != nil {
		return Project{}, err
	}

	_ = s.sink.Record(ctx, userID, "project_added", map[string]any{
		"project_id": project.ID,
		"name":       project.Name,
	})

	if projectURL != nil {
		s.pool.Submit(project.ID, *projectURL, s.resolvePreview)
	}

	return project, nil
}

// resolvePreview is the work a pool worker performs for one project's
// preview: call fetchsafe.FetchPreview (via the injected fetch function,
// with the worker pool's own background context — never the original
// request's context, which is already gone by the time this runs), then
// persist the outcome via UpdatePreview and record
// project_preview_resolved. Verbatim replication of
// ideas.Service.resolvePreview's rules (tasks.md T-04), including the
// "partial with zero recovered fields -> failed" rule: a partial result
// carrying literally nothing recovered is indistinguishable from a full
// fallback to the user, so it renders as failed rather than an empty
// "complete" state.
func (s *Service) resolvePreview(ctx context.Context, projectID int64, rawURL string) {
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
		if title == nil && imageURL == nil && description == nil {
			status = PreviewFailed
		}
	} else {
		slog.Debug("fetchsafe: project preview resolution failed", "project_id", projectID, "url", rawURL, "error", err)
	}

	if updateErr := s.repo.UpdatePreview(ctx, projectID, title, imageURL, description, status); updateErr != nil {
		// The project may have been deleted while the scrape was in flight
		// — UpdatePreview affecting zero rows is not surfaced as an error by
		// the repository, so any error reaching here is a genuine
		// unexpected failure worth logging.
		slog.Error("projects: failed to persist preview resolution", "project_id", projectID, "error", updateErr)
		return
	}

	_ = s.sink.Record(ctx, 0, "project_preview_resolved", map[string]any{
		"project_id": projectID,
		"status":     status,
	})
}

// knownStates enumerates the three valid values — any of the three is
// always a valid move from any of the others (AC-09); there is no
// forbidden-transition state machine to enforce.
func isKnownState(s State) bool {
	return s == StateIdea || s == StateDecidit || s == StateFet
}

// ChangeState moves a project to an absolute state, in any direction
// (design.md §5 Flux 2; covers AC-02, AC-03, AC-05 base, AC-07 base,
// AC-09, EC-05, EC-12, EC-13, NFR-01).
//
// The prior state used for the event's "from" field comes back from
// repo.UpdateState itself, read inside the same BEGIN IMMEDIATE
// transaction as the UPDATE (F-23 fix) — this method deliberately does
// NOT call repo.Get first. A separate, non-transactional read-then-write
// is a check-then-act race: under concurrent ChangeState calls on the
// same id, the losing request's separately-read "previous" value can
// already be stale by the time its own UpdateState commits, corrupting
// the project_state_changed audit trail NFR-01 depends on (the same
// defect class already fixed once on NIU-1/F-02 and NIU-4/F-01).
func (s *Service) ChangeState(ctx context.Context, userID, id int64, newState State) (Project, error) {
	if !isKnownState(newState) {
		return Project{}, ErrValidation{
			Code:    ValidationInvalidState,
			Message: "Estat no vàlid.",
		}
	}

	project, previousState, err := s.repo.UpdateState(ctx, id, userID, newState)
	if err != nil {
		return Project{}, err
	}

	_ = s.sink.Record(ctx, userID, "project_state_changed", map[string]any{
		"project_id": project.ID,
		"from":       string(previousState),
		"to":         string(project.State),
	})

	return project, nil
}

// Delete removes a project, idempotently (EC-13): a second call on an
// already-deleted id also succeeds without error and without writing a
// second event.
func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	existed, err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	if existed {
		_ = s.sink.Record(ctx, userID, "project_deleted", map[string]any{
			"project_id": id,
		})
	}
	return nil
}

// List returns every project — a single query, no N+1 (AC-06 base).
func (s *Service) List(ctx context.Context) ([]Project, error) {
	return s.repo.List(ctx)
}
