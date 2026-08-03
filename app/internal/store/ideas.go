package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"niu/internal/ideas"
)

// IdeasRepository implements ideas.Repository and ideas.EventSink on top
// of SQLite. All queries use bound parameters (?) — never fmt.Sprintf
// into SQL (design.md §9, NFR-02, EC-12).
type IdeasRepository struct {
	db *sql.DB
}

// NewIdeasRepository constructs an IdeasRepository.
func NewIdeasRepository(db *sql.DB) *IdeasRepository {
	return &IdeasRepository{db: db}
}

// Create inserts a new idea with preview_status='pending' and every
// preview field NULL (ADR-03 — every row is born pending).
func (r *IdeasRepository) Create(ctx context.Context, userID int64, url string) (ideas.Idea, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO activity_ideas (url, preview_status, added_by) VALUES (?, 'pending', ?)`,
		url, userID,
	)
	if err != nil {
		return ideas.Idea{}, fmt.Errorf("store: insert idea: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return ideas.Idea{}, fmt.Errorf("store: last insert id: %w", err)
	}

	return r.Get(ctx, id)
}

const ideaSelectColumns = `
	i.id, i.url, i.title, i.image_url, i.description, i.preview_status,
	i.added_by, au.name, au.display_name, au.avatar_emoji,
	i.created_at
`

const ideaSelectFrom = `
	FROM activity_ideas i
	LEFT JOIN users au ON au.id = i.added_by
`

func scanIdea(scan func(dest ...any) error) (ideas.Idea, error) {
	var (
		idea                                      ideas.Idea
		title, imageURL, description              sql.NullString
		previewStatus                             string
		addedByID                                 sql.NullInt64
		addedByName, addedByDisplay, addedByEmoji sql.NullString
		createdAt                                 time.Time
	)

	if err := scan(
		&idea.ID, &idea.URL, &title, &imageURL, &description, &previewStatus,
		&addedByID, &addedByName, &addedByDisplay, &addedByEmoji,
		&createdAt,
	); err != nil {
		return ideas.Idea{}, err
	}

	idea.PreviewStatus = ideas.PreviewStatus(previewStatus)
	idea.CreatedAt = createdAt

	if title.Valid {
		v := title.String
		idea.Title = &v
	}
	if imageURL.Valid {
		v := imageURL.String
		idea.ImageURL = &v
	}
	if description.Valid {
		v := description.String
		idea.Description = &v
	}
	if addedByID.Valid {
		idea.AddedBy = &ideas.User{
			ID:          addedByID.Int64,
			Name:        addedByName.String,
			DisplayName: addedByDisplay.String,
			AvatarEmoji: addedByEmoji.String,
		}
	}

	return idea, nil
}

// Get returns a single idea by ID.
func (r *IdeasRepository) Get(ctx context.Context, id int64) (ideas.Idea, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+ideaSelectColumns+ideaSelectFrom+` WHERE i.id = ?`, id)
	idea, err := scanIdea(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return ideas.Idea{}, ideas.ErrNotFound{ID: id}
	}
	if err != nil {
		return ideas.Idea{}, fmt.Errorf("store: get idea: %w", err)
	}
	return idea, nil
}

// List returns all ideas — a single query with a join to users, no N+1
// (design.md §5 Flux 2, NFR-09 — never triggers a re-scrape).
func (r *IdeasRepository) List(ctx context.Context) ([]ideas.Idea, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+ideaSelectColumns+ideaSelectFrom+` ORDER BY i.created_at DESC, i.id DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list ideas: %w", err)
	}
	defer rows.Close()

	var result []ideas.Idea
	for rows.Next() {
		idea, err := scanIdea(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("store: scan idea: %w", err)
		}
		result = append(result, idea)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list ideas rows: %w", err)
	}
	return result, nil
}

// UpdatePreview is the ONLY UPDATE allowed once a row has been created
// (design.md §4) — it never touches id/url/added_by/created_at. If id no
// longer exists (deleted while the scrape was in flight, design.md §5
// Flux 3), this affects zero rows and returns no error — the worker pool
// treats that as a normal, silent no-op.
func (r *IdeasRepository) UpdatePreview(ctx context.Context, id int64, title, imageURL, description *string, status ideas.PreviewStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE activity_ideas
		 SET title = ?, image_url = ?, description = ?, preview_status = ?
		 WHERE id = ?`,
		nullableString(title), nullableString(imageURL), nullableString(description), string(status), id,
	)
	if err != nil {
		return fmt.Errorf("store: update idea preview: %w", err)
	}
	return nil
}

// Delete removes the row. Idempotent: deleting an id that does not exist
// returns existed=false and no error (EC-15).
func (r *IdeasRepository) Delete(ctx context.Context, id int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM activity_ideas WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("store: delete idea: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("store: rows affected: %w", err)
	}
	return n > 0, nil
}

// Record writes one event to the append-only events table (implements
// ideas.EventSink) — delegates to the same table items.EventSink/
// projects.EventSink already use, no new column or table. userID may be 0
// when called from the background worker pool resolving a scrape
// (idea_preview_resolved is not attributable to either user).
func (r *IdeasRepository) Record(ctx context.Context, userID int64, kind string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("store: marshal event payload: %w", err)
	}
	var userIDArg any
	if userID != 0 {
		userIDArg = userID
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO events (user_id, kind, payload) VALUES (?, ?, ?)`,
		userIDArg, kind, string(body),
	)
	if err != nil {
		return fmt.Errorf("store: insert event: %w", err)
	}
	return nil
}
