package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"niu/internal/items"
)

// ItemsRepository implements items.Repository and items.EventSink on top
// of PostgreSQL. All queries use bound parameters ($1, $2, ...) — never
// fmt.Sprintf into SQL (design.md §9, NFR-03, EC-10).
type ItemsRepository struct {
	db *sql.DB
}

// NewItemsRepository constructs an ItemsRepository.
func NewItemsRepository(db *sql.DB) *ItemsRepository {
	return &ItemsRepository{db: db}
}

// Create inserts a new item, checking name uniqueness inside the same
// transaction as the INSERT (ADR-02). The DB-level unique index on
// name_normalized is the final authority — a race between two concurrent
// creates with the same name is resolved by the index, not just by the
// pre-check.
func (r *ItemsRepository) Create(ctx context.Context, userID int64, name, nameNormalized string) (items.Item, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return items.Item{}, fmt.Errorf("store: begin create tx: %w", err)
	}
	defer tx.Rollback()

	var existingLocation string
	err = tx.QueryRowContext(ctx,
		`SELECT location FROM items WHERE name_normalized = $1`,
		nameNormalized,
	).Scan(&existingLocation)
	if err == nil {
		return items.Item{}, items.ErrDuplicate{ExistingLocation: items.Location(existingLocation)}
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return items.Item{}, fmt.Errorf("store: check duplicate: %w", err)
	}

	// Same lock as Update: the new item's position is MAX+1 of the shopping
	// box, so a concurrent create or move into it must wait for this one.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, itemsMoveLockID); err != nil {
		return items.Item{}, fmt.Errorf("store: lock positions: %w", err)
	}

	var maxPos sql.NullFloat64
	if err := tx.QueryRowContext(ctx,
		`SELECT MAX(position) FROM items WHERE location = $1`,
		string(items.LocationShopping),
	).Scan(&maxPos); err != nil {
		return items.Item{}, fmt.Errorf("store: max position: %w", err)
	}
	position := 1.0
	if maxPos.Valid {
		position = maxPos.Float64 + 1.0
	}

	var id int64
	err = tx.QueryRowContext(ctx,
		`INSERT INTO items (name, name_normalized, location, position, added_by)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		name, nameNormalized, string(items.LocationShopping), position, userID,
	).Scan(&id)
	if err != nil {
		// A concurrent insert may have won the race against our pre-check;
		// the unique index surfaces as a constraint error here. PostgreSQL
		// aborts the transaction on any error, so the winner's location is
		// read outside it.
		if IsUniqueViolation(err) {
			_ = tx.Rollback()
			var loc string
			_ = r.db.QueryRowContext(ctx,
				`SELECT location FROM items WHERE name_normalized = $1`, nameNormalized,
			).Scan(&loc)
			return items.Item{}, items.ErrDuplicate{ExistingLocation: items.Location(loc)}
		}
		return items.Item{}, fmt.Errorf("store: insert item: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return items.Item{}, fmt.Errorf("store: commit create tx: %w", err)
	}

	return r.Get(ctx, id)
}

const itemSelectColumns = `
	i.id, i.name, i.location, i.position,
	i.added_by, au.name, au.display_name, au.avatar_emoji,
	i.moved_by, mu.name, mu.display_name, mu.avatar_emoji,
	i.moved_at, i.created_at, i.updated_at
`

const itemSelectFrom = `
	FROM items i
	LEFT JOIN users au ON au.id = i.added_by
	LEFT JOIN users mu ON mu.id = i.moved_by
`

func scanItem(scan func(dest ...any) error) (items.Item, error) {
	var (
		it                                        items.Item
		location                                  string
		addedByID                                 sql.NullInt64
		addedByName, addedByDisplay, addedByEmoji sql.NullString
		movedByID                                 sql.NullInt64
		movedByName, movedByDisplay, movedByEmoji sql.NullString
		movedAt                                   sql.NullTime
		createdAt, updatedAt                      time.Time
	)

	if err := scan(
		&it.ID, &it.Name, &location, &it.Position,
		&addedByID, &addedByName, &addedByDisplay, &addedByEmoji,
		&movedByID, &movedByName, &movedByDisplay, &movedByEmoji,
		&movedAt, &createdAt, &updatedAt,
	); err != nil {
		return items.Item{}, err
	}

	it.Location = items.Location(location)
	it.CreatedAt = createdAt
	it.UpdatedAt = updatedAt

	if addedByID.Valid {
		it.AddedBy = &items.User{
			ID:          addedByID.Int64,
			Name:        addedByName.String,
			DisplayName: addedByDisplay.String,
			AvatarEmoji: addedByEmoji.String,
		}
	}
	if movedByID.Valid {
		it.MovedBy = &items.User{
			ID:          movedByID.Int64,
			Name:        movedByName.String,
			DisplayName: movedByDisplay.String,
			AvatarEmoji: movedByEmoji.String,
		}
	}
	if movedAt.Valid {
		t := movedAt.Time
		it.MovedAt = &t
	}

	return it, nil
}

// Get returns a single item by ID.
func (r *ItemsRepository) Get(ctx context.Context, id int64) (items.Item, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+itemSelectColumns+itemSelectFrom+` WHERE i.id = $1`, id)
	it, err := scanItem(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return items.Item{}, items.ErrNotFound{ID: id}
	}
	if err != nil {
		return items.Item{}, fmt.Errorf("store: get item: %w", err)
	}
	return it, nil
}

// List returns all items ordered by location, position — a single query
// with joins to users, no N+1 (NFR-05).
func (r *ItemsRepository) List(ctx context.Context) ([]items.Item, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+itemSelectColumns+itemSelectFrom+` ORDER BY i.location, i.position`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list items: %w", err)
	}
	defer rows.Close()

	var result []items.Item
	for rows.Next() {
		it, err := scanItem(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("store: scan item: %w", err)
		}
		result = append(result, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list items rows: %w", err)
	}
	return result, nil
}

// Update applies a location move in a single transaction (ADR-01): last
// write wins by server timestamp, no optimistic locking / If-Match.
// The position argument is IGNORED — it is computed here, inside the
// transaction, from the destination's current MAX(position). ADR-01
// requires the move to be a single transaction: reading MAX(position)
// outside it and passing the result in would leave a window where a
// concurrent move to the same box computes the same position, and both
// rows land on top of each other. The parameter is kept so the Repository
// interface stays stable for callers that do not care.
// Concurrent moves are serialised with a transaction-scoped advisory lock
// (pg_advisory_xact_lock) taken before the position is read: a second mover
// waits for the first to commit and then reads the updated MAX(position), so
// two people moving items into the same box at the same moment can never land
// on the same position (AC-09). The lock is released at COMMIT/ROLLBACK.
func (r *ItemsRepository) Update(ctx context.Context, id, userID int64, newLocation items.Location, _ float64) (items.Item, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return items.Item{}, fmt.Errorf("store: begin move tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after commit

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, itemsMoveLockID); err != nil {
		return items.Item{}, fmt.Errorf("store: lock moves: %w", err)
	}

	// COALESCE covers the empty-destination case: first item lands at 1.0.
	var nextPosition float64
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(position), 0) + 1.0 FROM items WHERE location = $1`,
		string(newLocation),
	).Scan(&nextPosition); err != nil {
		return items.Item{}, fmt.Errorf("store: next position: %w", err)
	}

	res, err := tx.ExecContext(ctx,
		`UPDATE items
		 SET location = $1, moved_by = $2, moved_at = now(),
		     updated_at = now(), position = $3
		 WHERE id = $4`,
		string(newLocation), userID, nextPosition, id,
	)
	if err != nil {
		return items.Item{}, fmt.Errorf("store: update item: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return items.Item{}, fmt.Errorf("store: rows affected: %w", err)
	}
	if n == 0 {
		return items.Item{}, items.ErrNotFound{ID: id}
	}

	if err := tx.Commit(); err != nil {
		return items.Item{}, fmt.Errorf("store: commit move: %w", err)
	}
	return r.Get(ctx, id)
}

// itemsMoveLockID is the advisory-lock key that serialises the writes that
// compute a position (Create and Update).
const itemsMoveLockID = 7_310_245

// Delete removes the row. Idempotent: deleting an id that does not exist
// returns existed=false and no error (EC-11).
func (r *ItemsRepository) Delete(ctx context.Context, id int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM items WHERE id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("store: delete item: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("store: rows affected: %w", err)
	}
	return n > 0, nil
}

// ExistsByNormalizedName reports whether an item with this normalized
// name already exists, and if so, in which location.
func (r *ItemsRepository) ExistsByNormalizedName(ctx context.Context, nameNormalized string) (bool, items.Location, error) {
	var location string
	err := r.db.QueryRowContext(ctx,
		`SELECT location FROM items WHERE name_normalized = $1`, nameNormalized,
	).Scan(&location)
	if errors.Is(err, sql.ErrNoRows) {
		return false, "", nil
	}
	if err != nil {
		return false, "", fmt.Errorf("store: exists by normalized name: %w", err)
	}
	return true, items.Location(location), nil
}

// MaxPosition returns the highest position value in use for the given
// location, and whether that location currently has any items.
func (r *ItemsRepository) MaxPosition(ctx context.Context, location items.Location) (float64, bool, error) {
	var maxPos sql.NullFloat64
	if err := r.db.QueryRowContext(ctx,
		`SELECT MAX(position) FROM items WHERE location = $1`, string(location),
	).Scan(&maxPos); err != nil {
		return 0, false, fmt.Errorf("store: max position: %w", err)
	}
	if !maxPos.Valid {
		return 0, false, nil
	}
	return maxPos.Float64, true, nil
}

// Record writes one event to the append-only events table (implements
// items.EventSink).
func (r *ItemsRepository) Record(ctx context.Context, userID int64, kind string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("store: marshal event payload: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO events (user_id, kind, payload) VALUES ($1, $2, $3)`,
		userID, kind, string(body),
	)
	if err != nil {
		return fmt.Errorf("store: insert event: %w", err)
	}
	return nil
}

// GetUser resolves a user by ID (implements items.UserLookup).
func (r *ItemsRepository) GetUser(ctx context.Context, id int64) (items.User, error) {
	var u items.User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, display_name, avatar_emoji FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Name, &u.DisplayName, &u.AvatarEmoji)
	if err != nil {
		return items.User{}, fmt.Errorf("store: get user: %w", err)
	}
	return u, nil
}
