package items

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxNameLength = 200

// hasControlChars reports whether s contains any character that must not
// be stored in an item name (EC-05).
//
// This deliberately goes well beyond the ASCII control range. An earlier
// version checked only [\x00-\x08\x0B\x0C\x0E-\x1F], which let three
// classes of hostile input through:
//
//  1. Embedded \n, \r and \t. TrimSpace only strips them at the ends, so
//     "Llet\n\n\nPa" was stored as a single item rendering across lines.
//  2. Bidirectional overrides (U+202A-U+202E, U+2066-U+2069). These make
//     stored text render differently from what it is — "Comprar
//     <U+202E>selpmà 100" displays reversed, so what the other person
//     reads is not what is in the database. This is Trojan Source
//     (CVE-2021-42574).
//  3. Zero-width characters (U+200B-U+200D, U+FEFF). Invisible, so
//     "po<U+200B>ma" looks identical to "poma" but normalises
//     differently — a one-character bypass of the EC-06 duplicate rule.
//
// Unicode format characters (category Cf) cover the bidi and zero-width
// cases; unicode.IsControl covers C0 and C1. Neither catches the other,
// so both are needed.
func hasControlChars(s string) bool {
	for _, r := range s {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			// Rejected anywhere, including mid-string. TrimSpace has
			// already removed them from the ends by this point, so
			// anything left here is embedded.
			return true
		case unicode.IsControl(r):
			// C0 (U+0000-U+001F) and C1 (U+0080-U+009F).
			return true
		case unicode.Is(unicode.Cf, r):
			// Format characters: bidi overrides, zero-width joiners,
			// BOM. Note this also excludes U+200D, which some emoji
			// sequences use to join glyphs — acceptable here, since an
			// item name is short text, not a rich emoji composition.
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
	// Position is computed by the repository inside the move transaction
	// (ADR-01). Reading MAX(position) here and passing it down would put
	// the read outside the transaction, so two concurrent moves into the
	// same box could pick the same position. The zero passed below is a
	// placeholder the repository ignores.
	item, err := s.repo.Update(ctx, id, userID, newLocation, 0)
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
