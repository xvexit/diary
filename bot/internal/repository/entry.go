package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/dnevnik-bot/internal/model"
)

type Entry interface {
	Create(ctx context.Context, userID int64, content string) (*model.Entry, error)
	GetByID(ctx context.Context, id int) (*model.Entry, error)
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*model.Entry, error)
	CountByUser(ctx context.Context, userID int64) (int, error)
	Update(ctx context.Context, id int, content string) error
	Delete(ctx context.Context, id int) error
	RandomByUser(ctx context.Context, userID int64) (*model.Entry, error)
}

type PgEntry struct {
	pool *pgxpool.Pool
}

func NewPgEntry(pool *pgxpool.Pool) *PgEntry {
	return &PgEntry{pool: pool}
}

func (r *PgEntry) Create(ctx context.Context, userID int64, content string) (*model.Entry, error) {
	var e model.Entry
	err := r.pool.QueryRow(ctx,
		`INSERT INTO entries (user_id, content) VALUES ($1, $2)
		 RETURNING id, user_id, content, created_at, updated_at`,
		userID, content,
	).Scan(&e.ID, &e.UserID, &e.Content, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *PgEntry) GetByID(ctx context.Context, id int) (*model.Entry, error) {
	var e model.Entry
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, content, created_at, updated_at FROM entries WHERE id = $1`, id,
	).Scan(&e.ID, &e.UserID, &e.Content, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *PgEntry) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*model.Entry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, content, created_at, updated_at FROM entries
		 WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*model.Entry
	for rows.Next() {
		var e model.Entry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Content, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, &e)
	}
	return entries, nil
}

func (r *PgEntry) CountByUser(ctx context.Context, userID int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM entries WHERE user_id = $1`, userID,
	).Scan(&count)
	return count, err
}

func (r *PgEntry) Update(ctx context.Context, id int, content string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE entries SET content = $1, updated_at = $2 WHERE id = $3`,
		content, time.Now(), id,
	)
	return err
}

func (r *PgEntry) Delete(ctx context.Context, id int) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM entries WHERE id = $1`, id)
	return err
}

func (r *PgEntry) RandomByUser(ctx context.Context, userID int64) (*model.Entry, error) {
	var e model.Entry
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, content, created_at, updated_at FROM entries
		 WHERE user_id = $1 ORDER BY RANDOM() LIMIT 1`, userID,
	).Scan(&e.ID, &e.UserID, &e.Content, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}
