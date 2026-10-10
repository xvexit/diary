package service

import (
	"context"
	"errors"
	"strings"

	"github.com/user/dnevnik-bot/internal/model"
	"github.com/user/dnevnik-bot/internal/plugin"
)

var (
	ErrEmptyContent = errors.New("content cannot be empty")
	ErrNotFound     = errors.New("entry not found")
	ErrForbidden    = errors.New("access denied")
)

// EntryService routes every operation to the store of the diary it belongs to.
type EntryService struct {
	registry *plugin.Registry
}

func NewEntryService(registry *plugin.Registry) *EntryService {
	return &EntryService{registry: registry}
}

func (s *EntryService) store(slug string) (plugin.DiaryPlugin, bool) {
	return s.registry.Get(slug)
}

// buildContent produces the plain-text representation stored in the diary's
// content column. It is used for search and backwards-compatible rendering.
func buildContent(fields []model.EntryField) string {
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		if v := strings.TrimSpace(f.Value); v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, "\n")
}

func (s *EntryService) Create(ctx context.Context, userID int64, slug string, fields []model.EntryField) (*model.Entry, error) {
	p, ok := s.store(slug)
	if !ok {
		return nil, ErrUnknownPlugin
	}
	if err := p.Validate(fields); err != nil {
		return nil, err
	}
	content := buildContent(fields)
	if content == "" {
		return nil, ErrEmptyContent
	}
	return p.Store().Create(ctx, userID, content, fields)
}

func (s *EntryService) GetByID(ctx context.Context, userID int64, slug string, id int) (*model.Entry, error) {
	p, ok := s.store(slug)
	if !ok {
		return nil, ErrUnknownPlugin
	}
	e, err := p.Store().Get(ctx, userID, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return e, nil
}

func (s *EntryService) List(ctx context.Context, userID int64, slug string, page, pageSize int) ([]*model.Entry, int, error) {
	p, ok := s.store(slug)
	if !ok {
		return nil, 0, ErrUnknownPlugin
	}
	offset := (page - 1) * pageSize
	entries, err := p.Store().List(ctx, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := p.Store().Count(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

func (s *EntryService) Update(ctx context.Context, userID int64, slug string, id int, fields []model.EntryField) error {
	p, ok := s.store(slug)
	if !ok {
		return ErrUnknownPlugin
	}
	if _, err := p.Store().Get(ctx, userID, id); err != nil {
		return ErrNotFound
	}
	if err := p.Validate(fields); err != nil {
		return err
	}
	content := buildContent(fields)
	if content == "" {
		return ErrEmptyContent
	}
	return p.Store().Update(ctx, userID, id, content, fields)
}

func (s *EntryService) Delete(ctx context.Context, userID int64, slug string, id int) error {
	p, ok := s.store(slug)
	if !ok {
		return ErrUnknownPlugin
	}
	if _, err := p.Store().Get(ctx, userID, id); err != nil {
		return ErrNotFound
	}
	return p.Store().Delete(ctx, userID, id)
}

func (s *EntryService) Search(ctx context.Context, userID int64, slug, query string, page, pageSize int) ([]*model.Entry, int, error) {
	p, ok := s.store(slug)
	if !ok {
		return nil, 0, ErrUnknownPlugin
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, 0, ErrEmptyContent
	}
	offset := (page - 1) * pageSize
	return p.Store().Search(ctx, userID, query, pageSize, offset)
}

func (s *EntryService) Random(ctx context.Context, userID int64, slug string) (*model.Entry, error) {
	p, ok := s.store(slug)
	if !ok {
		return nil, ErrUnknownPlugin
	}
	return p.Store().Random(ctx, userID)
}
