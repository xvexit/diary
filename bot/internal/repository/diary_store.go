package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/dnevnik-bot/internal/model"
)

const diaryColumns = `id, user_id, content, fields, created_at, updated_at`

// PgDiaryStore is the default per-diary storage backed by a dedicated table
// (one table per diary). All diary tables share the same generic shape; a diary
// can later swap this out for a typed store with custom columns.
type PgDiaryStore struct {
	pool  *pgxpool.Pool
	table string
}

func NewDiaryStore(pool *pgxpool.Pool, table string) *PgDiaryStore {
	return &PgDiaryStore{pool: pool, table: table}
}

func (r *PgDiaryStore) TableName() string { return r.table }

type rowScanner interface {
	Scan(dest ...any) error
}

func marshalFields(fields []model.EntryField) string {
	if fields == nil {
		fields = []model.EntryField{}
	}
	b, err := json.Marshal(fields)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func scanEntry(s rowScanner) (*model.Entry, error) {
	var e model.Entry
	var raw []byte
	if err := s.Scan(&e.ID, &e.UserID, &e.Content, &raw, &e.CreatedAt, &e.UpdatedAt); err != nil {
		return nil, err
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &e.Fields); err != nil {
			return nil, err
		}
	}
	if e.Fields == nil {
		e.Fields = []model.EntryField{}
	}
	return &e, nil
}

func (r *PgDiaryStore) Create(ctx context.Context, userID int64, content string, fields []model.EntryField) (*model.Entry, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO `+r.table+` (user_id, content, fields)
		 VALUES ($1, $2, $3::jsonb)
		 RETURNING `+diaryColumns,
		userID, content, marshalFields(fields),
	)
	return scanEntry(row)
}

func (r *PgDiaryStore) Get(ctx context.Context, userID int64, id int) (*model.Entry, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+diaryColumns+` FROM `+r.table+` WHERE id = $1 AND user_id = $2`, id, userID,
	)
	return scanEntry(row)
}

func (r *PgDiaryStore) List(ctx context.Context, userID int64, limit, offset int) ([]*model.Entry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+diaryColumns+` FROM `+r.table+`
		 WHERE user_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*model.Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *PgDiaryStore) Count(ctx context.Context, userID int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM `+r.table+` WHERE user_id = $1`, userID,
	).Scan(&count)
	return count, err
}

func (r *PgDiaryStore) Update(ctx context.Context, userID int64, id int, content string, fields []model.EntryField) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE `+r.table+` SET content = $1, fields = $2::jsonb, updated_at = NOW()
		 WHERE id = $3 AND user_id = $4`,
		content, marshalFields(fields), id, userID,
	)
	return err
}

func (r *PgDiaryStore) Delete(ctx context.Context, userID int64, id int) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM `+r.table+` WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (r *PgDiaryStore) Search(ctx context.Context, userID int64, query string, limit, offset int) ([]*model.Entry, int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM `+r.table+` WHERE user_id = $1 AND content ILIKE '%' || $2 || '%'`,
		userID, query,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT `+diaryColumns+` FROM `+r.table+`
		 WHERE user_id = $1 AND content ILIKE '%' || $2 || '%'
		 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		userID, query, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var entries []*model.Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, 0, err
		}
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}

func (r *PgDiaryStore) Random(ctx context.Context, userID int64) (*model.Entry, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+diaryColumns+` FROM `+r.table+`
		 WHERE user_id = $1
		 ORDER BY RANDOM() LIMIT 1`,
		userID,
	)
	return scanEntry(row)
}
