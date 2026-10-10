package service

import (
	"context"
	"errors"

	"github.com/user/dnevnik-bot/internal/model"
	"github.com/user/dnevnik-bot/internal/plugin"
	"github.com/user/dnevnik-bot/internal/repository"
)

var (
	ErrUnknownPlugin = errors.New("неизвестный дневник")
	ErrBuiltinPlugin = errors.New("обычный дневник нельзя убрать")
)

// CatalogItem is a diary shown in the catalogue together with whether the user
// already added it.
type CatalogItem struct {
	Diary     plugin.DiaryPlugin
	Installed bool
}

// DiaryService owns the diary catalogue and which diaries a user has added.
// The default (free-form) diary is never part of the catalogue nor of the
// user's added list: it always exists implicitly.
type DiaryService struct {
	registry *plugin.Registry
	repo     repository.PluginRepo
}

func NewDiaryService(registry *plugin.Registry, repo repository.PluginRepo) *DiaryService {
	return &DiaryService{registry: registry, repo: repo}
}

// Default returns the always-present free-form diary.
func (s *DiaryService) Default() (plugin.DiaryPlugin, bool) {
	return s.registry.Get(model.DefaultPluginSlug)
}

func (s *DiaryService) Get(slug string) (plugin.DiaryPlugin, bool) {
	return s.registry.Get(slug)
}

// Catalog lists every diary except the default one, marking which the user has
// already added.
func (s *DiaryService) Catalog(ctx context.Context, userID int64) ([]CatalogItem, error) {
	installed, err := s.installedSet(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]CatalogItem, 0)
	for _, p := range s.registry.All() {
		if p.Metadata().Slug == model.DefaultPluginSlug {
			continue
		}
		out = append(out, CatalogItem{Diary: p, Installed: installed[p.Metadata().Slug]})
	}
	return out, nil
}

// Added returns the diaries the user added (the default one excluded).
func (s *DiaryService) Added(ctx context.Context, userID int64) ([]plugin.DiaryPlugin, error) {
	slugs, err := s.repo.ListInstalled(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]plugin.DiaryPlugin, 0, len(slugs))
	for _, slug := range slugs {
		if slug == model.DefaultPluginSlug {
			continue
		}
		if p, ok := s.registry.Get(slug); ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *DiaryService) Install(ctx context.Context, userID int64, slug string) error {
	if slug == model.DefaultPluginSlug {
		return ErrBuiltinPlugin
	}
	if _, ok := s.registry.Get(slug); !ok {
		return ErrUnknownPlugin
	}
	return s.repo.InstallPlugin(ctx, userID, slug)
}

func (s *DiaryService) Uninstall(ctx context.Context, userID int64, slug string) error {
	if slug == model.DefaultPluginSlug {
		return ErrBuiltinPlugin
	}
	if _, ok := s.registry.Get(slug); !ok {
		return ErrUnknownPlugin
	}
	return s.repo.UninstallPlugin(ctx, userID, slug)
}

func (s *DiaryService) installedSet(ctx context.Context, userID int64) (map[string]bool, error) {
	slugs, err := s.repo.ListInstalled(ctx, userID)
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(slugs))
	for _, slug := range slugs {
		set[slug] = true
	}
	return set, nil
}
