package projects

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
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
// rules. It depends only on the Repository and EventSink interfaces —
// never on database/sql or net/http (design.md §4).
type Service struct {
	repo Repository
	sink EventSink
}

// NewService constructs a Service.
func NewService(repo Repository, sink EventSink) *Service {
	return &Service{repo: repo, sink: sink}
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

// Add validates and creates a new project in state "idea" (design.md §5
// Flux 1; covers AC-01, AC-10, AC-14, AC-15, EC-01, EC-02, EC-03, EC-07,
// EC-16, EC-17).
func (s *Service) Add(ctx context.Context, userID int64, rawName string, rawBudget, rawTargetDate *string) (Project, error) {
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

	nameNormalized := NormalizeName(name)

	// Duplicate check + INSERT happen inside the same transaction at the
	// store layer (ADR-02), across ALL states (EC-03) — Create is
	// responsible for that atomicity and for surfacing ErrDuplicate when
	// the DB-level unique index rejects the insert.
	project, err := s.repo.Create(ctx, userID, name, nameNormalized, budget, targetDate)
	if err != nil {
		return Project{}, err
	}

	_ = s.sink.Record(ctx, userID, "project_added", map[string]any{
		"project_id": project.ID,
		"name":       project.Name,
	})

	return project, nil
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
func (s *Service) ChangeState(ctx context.Context, userID, id int64, newState State) (Project, error) {
	if !isKnownState(newState) {
		return Project{}, ErrValidation{
			Code:    ValidationInvalidState,
			Message: "Estat no vàlid.",
		}
	}

	previous, err := s.repo.Get(ctx, id)
	if err != nil {
		return Project{}, err
	}

	project, err := s.repo.UpdateState(ctx, id, userID, newState)
	if err != nil {
		return Project{}, err
	}

	_ = s.sink.Record(ctx, userID, "project_state_changed", map[string]any{
		"project_id": project.ID,
		"from":       string(previous.State),
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
