package builtin

import (
	"errors"
	"strings"

	"github.com/user/dnevnik-bot/internal/model"
	"github.com/user/dnevnik-bot/internal/plugin"
)

const SlugFreetext = "freetext"

// FreeText is the built-in free-form diary. It mirrors the original behaviour
// of the bot and keeps historical plain-text entries working.
type FreeText struct {
	store plugin.EntryStore
}

func NewFreeText(store plugin.EntryStore) *FreeText { return &FreeText{store: store} }

func (p *FreeText) Store() plugin.EntryStore { return p.store }

func (p *FreeText) Metadata() model.DiaryPluginMeta {
	return model.DiaryPluginMeta{
		Slug:        SlugFreetext,
		Name:        "свободный дневник",
		Description: "обычные записи свободным текстом.",
		Icon:        "📝",
	}
}

func (p *FreeText) Fields() []model.FieldSpec {
	return []model.FieldSpec{
		{
			Name:     "text",
			Prompt:   "напиши всё, что хочешь сохранить",
			Type:     model.FieldTypeText,
			Required: true,
		},
	}
}

func (p *FreeText) Validate(fields []model.EntryField) error {
	if strings.TrimSpace(plugin.FieldValue(fields, "text")) == "" {
		return errors.New("запись не может быть пустой")
	}
	return nil
}

func (p *FreeText) FormatEntry(fields []model.EntryField) string {
	return plugin.EscapeHTML(plugin.FieldValue(fields, "text"))
}

func (p *FreeText) FormatPreview(fields []model.EntryField) string {
	return plugin.Truncate(plugin.FieldValue(fields, "text"), 20)
}
