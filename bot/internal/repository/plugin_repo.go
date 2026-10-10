package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PluginRepo interface {
	InstallPlugin(ctx context.Context, userID int64, slug string) error
	UninstallPlugin(ctx context.Context, userID int64, slug string) error
	ListInstalled(ctx context.Context, userID int64) ([]string, error)
	IsInstalled(ctx context.Context, userID int64, slug string) (bool, error)
}

type PgPluginRepo struct {
	pool *pgxpool.Pool
}

func NewPgPluginRepo(pool *pgxpool.Pool) *PgPluginRepo {
	return &PgPluginRepo{pool: pool}
}

func (r *PgPluginRepo) InstallPlugin(ctx context.Context, userID int64, slug string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_plugins (user_id, plugin_slug) VALUES ($1, $2)
		 ON CONFLICT (user_id, plugin_slug) DO NOTHING`,
		userID, slug,
	)
	return err
}

func (r *PgPluginRepo) UninstallPlugin(ctx context.Context, userID int64, slug string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM user_plugins WHERE user_id = $1 AND plugin_slug = $2`,
		userID, slug,
	)
	return err
}

func (r *PgPluginRepo) ListInstalled(ctx context.Context, userID int64) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT plugin_slug FROM user_plugins WHERE user_id = $1 ORDER BY installed_at`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slugs []string
	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			return nil, err
		}
		slugs = append(slugs, slug)
	}
	return slugs, rows.Err()
}

func (r *PgPluginRepo) IsInstalled(ctx context.Context, userID int64, slug string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (
		   SELECT 1 FROM user_plugins WHERE user_id = $1 AND plugin_slug = $2
		 )`,
		userID, slug,
	).Scan(&exists)
	return exists, err
}
