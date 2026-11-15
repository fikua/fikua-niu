package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"niu/internal/projects"
)

// ProjectsRepository implements projects.Repository and
// projects.EventSink on top of PostgreSQL. All queries use bound parameters
// ($1, $2, ...) — never fmt.Sprintf into SQL (design.md §9, NFR-03, EC-09).
type ProjectsRepository struct {
	db *sql.DB
}

// NewProjectsRepository constructs a ProjectsRepository.
func NewProjectsRepository(db *sql.DB) *ProjectsRepository {
	return &ProjectsRepository{db: db}
}

// Create inserts a new project, checking name uniqueness across ALL
// states inside the same transaction as the INSERT (ADR-02/EC-03). The
// DB-level unique index on name_normalized is the final authority — a
// race between two concurrent creates with the same name is resolved by
// the index, not just by the pre-check.
//
// url is optional (NIU-11): when non-nil, preview_status is set to
// 'pending' at INSERT time — mirroring IdeasRepository.Create, every row
// with a url is born pending. When nil, preview_status stays NULL (not
// 'pending' — migration 005 makes the column NULL-able precisely so a
// project without a url never shows a preview spinner that will never
// resolve).
func (r *ProjectsRepository) Create(ctx context.Context, userID int64, name, nameNormalized string, budget, targetDate, url *string) (projects.Project, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return projects.Project{}, fmt.Errorf("store: begin create project tx: %w", err)
	}
	defer tx.Rollback()

	var exists int
	err = tx.QueryRowContext(ctx,
		`SELECT 1 FROM projects WHERE name_normalized = $1`,
		nameNormalized,
	).Scan(&exists)
	if err == nil {
		return projects.Project{}, projects.ErrDuplicate{}
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return projects.Project{}, fmt.Errorf("store: check duplicate project: %w", err)
	}

	var previewStatus sql.NullString
	if url != nil {
		previewStatus = sql.NullString{String: projects.PreviewPending, Valid: true}
	}

	var id int64
	err = tx.QueryRowContext(ctx,
		`INSERT INTO projects (name, name_normalized, state, budget, target_date, url, preview_status, added_by, last_updated_by)
		 VALUES ($1, $2, 'idea', $3, $4, $5, $6, $7, $8) RETURNING id`,
		name, nameNormalized, nullableString(budget), nullableString(targetDate), nullableString(url), previewStatus, userID, userID,
	).Scan(&id)
	if err != nil {
		// A concurrent insert may have won the race against our pre-check;
		// the unique index surfaces as a constraint error here.
		if IsUniqueViolation(err) {
			return projects.Project{}, projects.ErrDuplicate{}
		}
		return projects.Project{}, fmt.Errorf("store: insert project: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return projects.Project{}, fmt.Errorf("store: commit create project tx: %w", err)
	}

	return r.Get(ctx, id)
}

func nullableString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

const projectSelectColumns = `
	p.id, p.name, p.state, p.budget, p.target_date,
	p.url, p.title, p.image_url, p.description, p.preview_status,
	p.added_by, au.name, au.display_name, au.avatar_emoji,
	p.last_updated_by, lu.name, lu.display_name, lu.avatar_emoji,
	p.created_at, p.updated_at
`

const projectSelectFrom = `
	FROM projects p
	LEFT JOIN users au ON au.id = p.added_by
	LEFT JOIN users lu ON lu.id = p.last_updated_by
`

func scanProject(scan func(dest ...any) error) (projects.Project, error) {
	var (
		p                                              projects.Project
		state                                          string
		budget, targetDate                             sql.NullString
		url, title, imageURL, description, previewStat sql.NullString
		addedByID                                      sql.NullInt64
		addedByName, addedByDisplay, addedByEmoji      sql.NullString
		luByID                                         sql.NullInt64
		luByName, luByDisplay, luByEmoji               sql.NullString
		createdAt, updatedAt                           time.Time
	)

	if err := scan(
		&p.ID, &p.Name, &state, &budget, &targetDate,
		&url, &title, &imageURL, &description, &previewStat,
		&addedByID, &addedByName, &addedByDisplay, &addedByEmoji,
		&luByID, &luByName, &luByDisplay, &luByEmoji,
		&createdAt, &updatedAt,
	); err != nil {
		return projects.Project{}, err
	}

	p.State = projects.State(state)
	p.CreatedAt = createdAt
	p.UpdatedAt = updatedAt

	if budget.Valid {
		v := budget.String
		p.Budget = &v
	}
	if targetDate.Valid {
		v := targetDate.String
		p.TargetDate = &v
	}
	if url.Valid {
		v := url.String
		p.URL = &v
	}
	if title.Valid {
		v := title.String
		p.Title = &v
	}
	if imageURL.Valid {
		v := imageURL.String
		p.ImageURL = &v
	}
	if description.Valid {
		v := description.String
		p.Description = &v
	}
	if previewStat.Valid {
		v := previewStat.String
		p.PreviewStatus = &v
	}

	if addedByID.Valid {
		p.AddedBy = &projects.User{
			ID:          addedByID.Int64,
			Name:        addedByName.String,
			DisplayName: addedByDisplay.String,
			AvatarEmoji: addedByEmoji.String,
		}
	}
	if luByID.Valid {
		p.LastUpdatedBy = &projects.User{
			ID:          luByID.Int64,
			Name:        luByName.String,
			DisplayName: luByDisplay.String,
			AvatarEmoji: luByEmoji.String,
		}
	}

	return p, nil
}

// Get returns a single project by ID.
func (r *ProjectsRepository) Get(ctx context.Context, id int64) (projects.Project, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+projectSelectColumns+projectSelectFrom+` WHERE p.id = $1`, id)
	p, err := scanProject(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return projects.Project{}, projects.ErrNotFound{ID: id}
	}
	if err != nil {
		return projects.Project{}, fmt.Errorf("store: get project: %w", err)
	}
	return p, nil
}

// List returns all projects — a single query with joins to users, no N+1.
func (r *ProjectsRepository) List(ctx context.Context) ([]projects.Project, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+projectSelectColumns+projectSelectFrom+` ORDER BY p.created_at, p.id`,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list projects: %w", err)
	}
	defer rows.Close()

	var result []projects.Project
	for rows.Next() {
		p, err := scanProject(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("store: scan project: %w", err)
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list projects rows: %w", err)
	}
	return result, nil
}

// UpdateState applies a state transition in a single transaction (ADR-01 of
// NIU-1, reapplied here — design.md §5 Flux 2): last write wins by server
// timestamp, no optimistic locking.
//
// The row is read with SELECT ... FOR UPDATE, so a concurrent UpdateState on
// the same project waits for this transaction to commit. That keeps both
// guarantees of the original design (F-23): the previous state is exactly the
// value this commit overwrote, and the returned Project — read inside the
// transaction, before COMMIT — is this transaction's write, never someone
// else's.
func (r *ProjectsRepository) UpdateState(ctx context.Context, id, userID int64, newState projects.State) (projects.Project, projects.State, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return projects.Project{}, "", fmt.Errorf("store: begin update state tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after commit

	var previousState string
	err = tx.QueryRowContext(ctx, `SELECT state FROM projects WHERE id = $1 FOR UPDATE`, id).Scan(&previousState)
	if errors.Is(err, sql.ErrNoRows) {
		return projects.Project{}, "", projects.ErrNotFound{ID: id}
	}
	if err != nil {
		return projects.Project{}, "", fmt.Errorf("store: read previous project state: %w", err)
	}

	res, err := tx.ExecContext(ctx,
		`UPDATE projects
		 SET state = $1, last_updated_by = $2, updated_at = now()
		 WHERE id = $3`,
		string(newState), userID, id,
	)
	if err != nil {
		return projects.Project{}, "", fmt.Errorf("store: update project state: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return projects.Project{}, "", fmt.Errorf("store: rows affected: %w", err)
	}
	if n == 0 {
		return projects.Project{}, "", projects.ErrNotFound{ID: id}
	}

	row := tx.QueryRowContext(ctx, `SELECT `+projectSelectColumns+projectSelectFrom+` WHERE p.id = $1`, id)
	updated, err := scanProject(row.Scan)
	if err != nil {
		return projects.Project{}, "", fmt.Errorf("store: read updated project: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return projects.Project{}, "", fmt.Errorf("store: commit update project state: %w", err)
	}
	return updated, projects.State(previousState), nil
}

// Delete removes the row. Idempotent: deleting an id that does not exist
// returns existed=false and no error (EC-13).
func (r *ProjectsRepository) Delete(ctx context.Context, id int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("store: delete project: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("store: rows affected: %w", err)
	}
	return n > 0, nil
}

// ExistsByNormalizedName reports whether a project with this normalized
// name already exists, in ANY state (EC-03) — a DELETEd project is not
// counted (EC-04, hard delete, no soft-delete column to exclude).
func (r *ProjectsRepository) ExistsByNormalizedName(ctx context.Context, nameNormalized string) (bool, error) {
	var exists int
	err := r.db.QueryRowContext(ctx,
		`SELECT 1 FROM projects WHERE name_normalized = $1`, nameNormalized,
	).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("store: exists by normalized name: %w", err)
	}
	return true, nil
}

// UpdatePreview is the ONLY UPDATE allowed for the preview fields (NIU-11
// — mirrors IdeasRepository.UpdatePreview) — it never touches
// id/name/state/budget/target_date/url. If id no longer exists (deleted
// while the scrape was in flight), this affects zero rows and returns no
// error — the worker pool treats that as a normal, silent no-op.
func (r *ProjectsRepository) UpdatePreview(ctx context.Context, id int64, title, imageURL, description *string, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE projects
		 SET title = $1, image_url = $2, description = $3, preview_status = $4
		 WHERE id = $5`,
		nullableString(title), nullableString(imageURL), nullableString(description), status, id,
	)
	if err != nil {
		return fmt.Errorf("store: update project preview: %w", err)
	}
	return nil
}

// Record writes one event to the append-only events table (implements
// projects.EventSink) — delegates to the same table items.EventSink uses,
// no new column or type.
func (r *ProjectsRepository) Record(ctx context.Context, userID int64, kind string, payload any) error {
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
