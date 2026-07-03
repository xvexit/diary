package service

import (
	"context"
	"errors"
	"strings"

	"github.com/user/dnevnik-bot/internal/model"
	"github.com/user/dnevnik-bot/internal/repository"
)

var (
	ErrEmptyContent = errors.New("content cannot be empty")
	ErrNotFound     = errors.New("entry not found")
	ErrForbidden    = errors.New("access denied")
)

type EntryService struct {
	repo repository.Entry
}

func NewEntryService(repo repository.Entry) *EntryService {
	return &EntryService{repo: repo}
}

func (s *EntryService) Create(ctx context.Context, userID int64, content string) (*model.Entry, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, ErrEmptyContent
	}
	return s.repo.Create(ctx, userID, content)
}

func (s *EntryService) GetByID(ctx context.Context, userID int64, id int) (*model.Entry, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	if e.UserID != userID {
		return nil, ErrForbidden
	}
	return e, nil
}

func (s *EntryService) ListByUser(ctx context.Context, userID int64, page, pageSize int) ([]*model.Entry, int, error) {
	offset := (page - 1) * pageSize
	entries, err := s.repo.ListByUser(ctx, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountByUser(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

func (s *EntryService) Update(ctx context.Context, userID int64, id int, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return ErrEmptyContent
	}
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	if e.UserID != userID {
		return ErrForbidden
	}
	return s.repo.Update(ctx, id, content)
}

func (s *EntryService) Delete(ctx context.Context, userID int64, id int) error {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	if e.UserID != userID {
		return ErrForbidden
	}
	return s.repo.Delete(ctx, id)
}

func (s *EntryService) RandomByUser(ctx context.Context, userID int64) (*model.Entry, error) {
	return s.repo.RandomByUser(ctx, userID)
}
