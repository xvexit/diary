package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserSettings struct {
	UserID          int64
	ReminderEnabled bool
	ReminderTime    string
}

type Settings interface {
	Get(ctx context.Context, userID int64) (*UserSettings, error)
	Upsert(ctx context.Context, s *UserSettings) error
	SetReminderEnabled(ctx context.Context, userID int64, enabled bool) error
	SetReminderTime(ctx context.Context, userID int64, t string) error
	GetUsersDueForReminder(ctx context.Context, now time.Time) ([]int64, error)
	HasEntryToday(ctx context.Context, userID int64) (bool, error)
}

type PgSettings struct {
	pool *pgxpool.Pool
}

func NewPgSettings(pool *pgxpool.Pool) *PgSettings {
	return &PgSettings{pool: pool}
}

func (r *PgSettings) Get(ctx context.Context, userID int64) (*UserSettings, error) {
	var s UserSettings
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, reminder_enabled, reminder_time::TEXT FROM user_settings WHERE user_id = $1`,
		userID,
	).Scan(&s.UserID, &s.ReminderEnabled, &s.ReminderTime)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PgSettings) Upsert(ctx context.Context, s *UserSettings) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_settings (user_id, reminder_enabled, reminder_time)
		 VALUES ($1, $2, $3::TIME)
		 ON CONFLICT (user_id) DO UPDATE SET reminder_enabled = $2, reminder_time = $3::TIME`,
		s.UserID, s.ReminderEnabled, s.ReminderTime,
	)
	return err
}

func (r *PgSettings) SetReminderEnabled(ctx context.Context, userID int64, enabled bool) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_settings (user_id, reminder_enabled, reminder_time)
		 VALUES ($1, $2, '21:00')
		 ON CONFLICT (user_id) DO UPDATE SET reminder_enabled = $2`,
		userID, enabled,
	)
	return err
}

func (r *PgSettings) SetReminderTime(ctx context.Context, userID int64, t string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_settings (user_id, reminder_enabled, reminder_time)
		 VALUES ($1, true, $2::TIME)
		 ON CONFLICT (user_id) DO UPDATE SET reminder_time = $2::TIME`,
		userID, t,
	)
	return err
}

func (r *PgSettings) GetUsersDueForReminder(ctx context.Context, now time.Time) ([]int64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT s.user_id FROM user_settings s
		 WHERE s.reminder_enabled = true
		 AND s.reminder_time <= $1::TIME
		 AND NOT EXISTS (
		   SELECT 1 FROM entries e
		   WHERE e.user_id = s.user_id
		   AND e.created_at::DATE = $1::DATE
		 )`,
		now,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *PgSettings) HasEntryToday(ctx context.Context, userID int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (
		   SELECT 1 FROM entries
		   WHERE user_id = $1 AND created_at::DATE = CURRENT_DATE
		 )`, userID,
	).Scan(&exists)
	return exists, err
}
