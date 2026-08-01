package items

import (
	"context"
	"strings"
	"unicode/utf8"
)

const maxNameLength = 200

// controlCharsPattern matches ASCII control bytes excluding common
// whitespace (tab/newline/CR are already handled by TrimSpace trimming
// leading/trailing space; embedded control bytes anywhere in the name are
// rejected per EC-05). Mirrors the reference regex in
// design-system/screen-desktop.html: [\x00-\x08\x0B\x0C\x0E-\x1F].
func hasControlChars(s string) bool {
	for _, r := range s {
		if (r >= 0x00 && r <= 0x08) || r == 0x0B || r == 0x0C || (r >= 0x0E && r <= 0x1F) {
			return true
		}
	}
	return false
}

// Service implements the shopping-list/pantry business rules. It depends
// only on the Repository and EventSink interfaces — never on
// database/sql or net/http (design.md §4).
type Service struct {
	repo  Repository
	sink  EventSink
	users UserLookup
}

// UserLookup resolves a userID to a full User for embedding in Item
// responses (added_by/moved_by). Implemented by internal/store.
type UserLookup interface {
	GetUser(ctx context.Context, id int64) (User, error)
}

// NewService constructs a Service.
func NewService(repo Repository, sink EventSink, users UserLookup) *Service {
	return &Service{repo: repo, sink: sink, users: users}
}

// Add validates and creates a new item in "shopping" (design.md §5 Flux 1;
// covers AC-01, EC-01..EC-07).
func (s *Service) Add(ctx context.Context, userID int64, rawName string) (Item, error) {
	trimmed := strings.TrimSpace(rawName)

	if trimmed == "" {
		return Item{}, ErrValidation{
			Code:    ValidationEmpty,
			Message: "Escriu un nom abans d'afegir.",
		}
	}

	if utf8.RuneCountInString(trimmed) > maxNameLength {
		return Item{}, ErrValidation{
			Code:    ValidationTooLong,
			Message: "Massa llarg — màxim 200 caràcters.",
		}
	}

	if hasControlChars(rawName) {
		return Item{}, ErrValidation{
			Code:    ValidationControlChars,
			Message: "Aquest nom conté caràcters no vàlids.",
		}
	}

	nameNormalized := NormalizeName(trimmed)

	// Duplicate check + INSERT happen inside the same transaction at the
	// store layer (ADR-02) — Create is responsible for that atomicity and
	// for surfacing ErrDuplicate when the DB-level unique index rejects
	// the insert.
	item, err := s.repo.Create(ctx, userID, trimmed, nameNormalized)
	if err != nil {
		return Item{}, err
	}

	_ = s.sink.Record(ctx, userID, "item_added", map[string]any{
		"item_id": item.ID,
		"name":    item.Name,
	})

	return item, nil
}

// Move updates an item's location (design.md §5 Flux 2; ADR-01; covers
// AC-02, AC-03, AC-04, AC-09, EC-12).
func (s *Service) Move(ctx context.Context, userID, id int64, newLocation Location) (Item, error) {
	maxPos, _, err := s.repo.MaxPosition(ctx, newLocation)
	if err != nil {
		return Item{}, err
	}
	newPosition := maxPos + 1.0

	item, err := s.repo.Update(ctx, id, userID, newLocation, newPosition)
	if err != nil {
		return Item{}, err
	}

	_ = s.sink.Record(ctx, userID, "item_moved", map[string]any{
		"item_id":  item.ID,
		"location": string(item.Location),
	})

	return item, nil
}

// Delete removes an item, idempotently (EC-11): a second call on an
// already-deleted id also succeeds without error and without writing a
// second event.
func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	existed, err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	if existed {
		_ = s.sink.Record(ctx, userID, "item_deleted", map[string]any{
			"item_id": id,
		})
	}
	return nil
}

// List returns every item across both boxes, ordered by location then
// position — a single query, no N+1 (AC-07 base, NFR-05 base).
func (s *Service) List(ctx context.Context) ([]Item, error) {
	return s.repo.List(ctx)
}

// CurrentUser resolves a userID (obtained by httpapi from
// auth.Authenticator) to the full User record.
func (s *Service) CurrentUser(ctx context.Context, userID int64) (User, error) {
	return s.users.GetUser(ctx, userID)
}
