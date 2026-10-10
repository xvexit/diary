package repository

import (
	"context"
	"strings"
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
	GetUsersDueForReminder(ctx context.Context) ([]int64, error)
	HasEntryToday(ctx context.Context, userID int64, date time.Time) (bool, error)
}

type PgSettings struct {
	pool   *pgxpool.Pool
	tables []string
}

func NewPgSettings(pool *pgxpool.Pool, diaryTables []string) *PgSettings {
	return &PgSettings{pool: pool, tables: diaryTables}
}

// entryExistsSQL builds a query testing whether any diary table has a row for
// the given user matching the date expression. userExpr references the user id
// (e.g. "$1" or "s.user_id"); dateExpr is a SQL date expression.
func (r *PgSettings) entryExistsSQL(userExpr, dateExpr string) string {
	if len(r.tables) == 0 {
		return "SELECT 1 WHERE false"
	}
	parts := make([]string, 0, len(r.tables))
	for _, t := range r.tables {
		parts = append(parts, "SELECT 1 FROM "+t+" WHERE user_id = "+userExpr+" AND created_at::DATE = "+dateExpr)
	}
	return strings.Join(parts, " UNION ALL ")
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

func (r *PgSettings) GetUsersDueForReminder(ctx context.Context) ([]int64, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT s.user_id FROM user_settings s
		 WHERE s.reminder_enabled = true
		 AND s.reminder_time <= LOCALTIME
		 AND NOT EXISTS (`+r.entryExistsSQL("s.user_id", "CURRENT_DATE")+`)`,
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

type ReminderUser struct {
	UserID       int64
	ReminderTime string
}

func (r *PgSettings) GetAllReminderUsers(ctx context.Context) ([]ReminderUser, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT user_id, reminder_time::TEXT FROM user_settings
		 WHERE reminder_enabled = true`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []ReminderUser
	for rows.Next() {
		var u ReminderUser
		if err := rows.Scan(&u.UserID, &u.ReminderTime); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *PgSettings) HasEntryToday(ctx context.Context, userID int64, date time.Time) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (`+r.entryExistsSQL("$1", "$2::DATE")+`)`, userID, date,
	).Scan(&exists)
	return exists, err
}
