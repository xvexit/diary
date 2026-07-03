package service

import (
	"context"
	"errors"
	"regexp"

	"github.com/user/dnevnik-bot/internal/repository"
)

var timeRe = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)

var ErrInvalidTime = errors.New("неверный формат времени, используй HH:MM (например 21:00)")

type SettingsService struct {
	repo repository.Settings
}

func NewSettingsService(repo repository.Settings) *SettingsService {
	return &SettingsService{repo: repo}
}

func (s *SettingsService) Get(ctx context.Context, userID int64) (*repository.UserSettings, error) {
	return s.repo.Get(ctx, userID)
}

func (s *SettingsService) ToggleReminder(ctx context.Context, userID int64) error {
	settings, err := s.repo.Get(ctx, userID)
	if err != nil {
		return s.repo.SetReminderEnabled(ctx, userID, true)
	}
	return s.repo.SetReminderEnabled(ctx, userID, !settings.ReminderEnabled)
}

func (s *SettingsService) SetReminderTime(ctx context.Context, userID int64, t string) error {
	if !timeRe.MatchString(t) {
		return ErrInvalidTime
	}
	return s.repo.SetReminderTime(ctx, userID, t)
}

func (s *SettingsService) Upsert(ctx context.Context, userID int64) error {
	return s.repo.Upsert(ctx, &repository.UserSettings{
		UserID:          userID,
		ReminderEnabled: true,
		ReminderTime:    "21:00",
	})
}
