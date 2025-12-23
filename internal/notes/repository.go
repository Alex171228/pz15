package notes

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Repository struct {
	db *sql.DB

	stmtGet    *sql.Stmt
	stmtUpdate *sql.Stmt
	stmtDelete *sql.Stmt
}

func NewRepository(ctx context.Context, db *sql.DB) (*Repository, error) {
	get, err := db.PrepareContext(ctx, `
		SELECT id, title, content, created_at
		FROM notes
		WHERE id = $1
	`)
	if err != nil {
		return nil, err
	}

	upd, err := db.PrepareContext(ctx, `
		UPDATE notes
		SET title = $1, content = $2
		WHERE id = $3
		RETURNING id, title, content, created_at
	`)
	if err != nil {
		return nil, err
	}

	del, err := db.PrepareContext(ctx, `DELETE FROM notes WHERE id = $1`)
	if err != nil {
		return nil, err
	}

	return &Repository{
		db:         db,
		stmtGet:    get,
		stmtUpdate: upd,
		stmtDelete: del,
	}, nil
}

func (r *Repository) Close() error {
	for _, s := range []*sql.Stmt{r.stmtGet, r.stmtUpdate, r.stmtDelete} {
		if s != nil {
			_ = s.Close()
		}
	}
	return nil
}

// Create uses explicit transaction: INSERT notes + INSERT audit.
func (r *Repository) Create(ctx context.Context, title, content string) (Note, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return Note{}, err
	}
	defer tx.Rollback()

	var n Note
	err = tx.QueryRowContext(ctx, `
		INSERT INTO notes (title, content) VALUES ($1, $2)
		RETURNING id, title, content, created_at
	`, title, content).Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt)
	if err != nil {
		return Note{}, err
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO notes_audit (note_id, action) VALUES ($1, $2)`, n.ID, "create")
	if err != nil {
		return Note{}, err
	}

	if err := tx.Commit(); err != nil {
		return Note{}, err
	}
	return n, nil
}

func (r *Repository) Get(ctx context.Context, id int64) (Note, error) {
	var n Note
	err := r.stmtGet.QueryRowContext(ctx, id).Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Note{}, sql.ErrNoRows
	}
	return n, err
}

func (r *Repository) Update(ctx context.Context, id int64, title, content string) (Note, error) {
	var n Note
	err := r.stmtUpdate.QueryRowContext(ctx, title, content, id).Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Note{}, sql.ErrNoRows
	}
	return n, err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	res, err := r.stmtDelete.ExecContext(ctx, id)
	if err != nil {
		return err
	}
	a, _ := res.RowsAffected()
	if a == 0 {
		return sql.ErrNoRows
	}
	return nil
}

type ListParams struct {
	Limit           int
	CursorCreatedAt *time.Time
	CursorID        *int64
	Query           string
}

func (r *Repository) List(ctx context.Context, p ListParams) ([]Note, error) {
	if p.Limit <= 0 || p.Limit > 200 {
		p.Limit = 20
	}

	// Search (GIN on to_tsvector(title))
	if p.Query != "" {
		rows, err := r.db.QueryContext(ctx, `
			SELECT id, title, content, created_at
			FROM notes
			WHERE to_tsvector('simple', title) @@ plainto_tsquery('simple', $1)
			ORDER BY created_at DESC, id DESC
			LIMIT $2
		`, p.Query, p.Limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanNotes(rows)
	}

	// Keyset pagination
	if p.CursorCreatedAt != nil && p.CursorID != nil {
		rows, err := r.db.QueryContext(ctx, `
			SELECT id, title, content, created_at
			FROM notes
			WHERE (created_at, id) < ($1, $2)
			ORDER BY created_at DESC, id DESC
			LIMIT $3
		`, *p.CursorCreatedAt, *p.CursorID, p.Limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanNotes(rows)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, content, created_at
		FROM notes
		ORDER BY created_at DESC, id DESC
		LIMIT $1
	`, p.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNotes(rows)
}

// BatchGet: один запрос вместо N запросов (ANY($1)).
func (r *Repository) BatchGet(ctx context.Context, ids []int64) ([]Note, error) {
	if len(ids) == 0 {
		return []Note{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, content, created_at
		FROM notes
		WHERE id = ANY($1)
		ORDER BY created_at DESC, id DESC
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNotes(rows)
}

func scanNotes(rows *sql.Rows) ([]Note, error) {
	out := make([]Note, 0, 32)
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
