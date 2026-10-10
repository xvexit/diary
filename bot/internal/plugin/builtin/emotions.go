package builtin

import (
	"errors"
	"strings"

	"github.com/user/dnevnik-bot/internal/model"
	"github.com/user/dnevnik-bot/internal/plugin"
)

const SlugEmotions = "emotions"

// Emotions is the built-in CBT-style thoughts & feelings diary.
type Emotions struct {
	store plugin.EntryStore
}

func NewEmotions(store plugin.EntryStore) *Emotions { return &Emotions{store: store} }

func (p *Emotions) Store() plugin.EntryStore { return p.store }

func (p *Emotions) Metadata() model.DiaryPluginMeta {
	return model.DiaryPluginMeta{
		Slug:        SlugEmotions,
		Name:        "мысли и эмоции",
		Description: "разбор ситуации, мыслей и эмоций (кпт).",
		Icon:        "🧠",
	}
}

func (p *Emotions) Fields() []model.FieldSpec {
	return []model.FieldSpec{
		{
			Name:     "situation",
			Prompt:   "опиши ситуацию",
			Type:     model.FieldTypeText,
			Required: true,
		},
		{
			Name:     "thoughts",
			Prompt:   "какие мысли возникли?",
			Type:     model.FieldTypeText,
			Required: true,
		},
		{
			Name:     "emotions",
			Prompt:   "выбери эмоции (можно несколько)",
			Type:     model.FieldTypeChoice,
			Options:  []string{"тревога", "грусть", "гнев", "радость", "вина", "стыд", "страх", "удивление"},
			Required: true,
			Multiple: true,
		},
		{
			Name:     "rational_response",
			Prompt:   "рациональный ответ на эти мысли",
			Type:     model.FieldTypeText,
			Required: false,
		},
	}
}

func (p *Emotions) Validate(fields []model.EntryField) error {
	if strings.TrimSpace(plugin.FieldValue(fields, "situation")) == "" {
		return errors.New("опишите ситуацию")
	}
	if strings.TrimSpace(plugin.FieldValue(fields, "thoughts")) == "" {
		return errors.New("опишите возникшие мысли")
	}
	if strings.TrimSpace(plugin.FieldValue(fields, "emotions")) == "" {
		return errors.New("выберите хотя бы одну эмоцию")
	}
	return nil
}

func (p *Emotions) FormatEntry(fields []model.EntryField) string {
	var b strings.Builder
	b.WriteString("<b>🧠 мысли и эмоции</b>\n\n")
	b.WriteString("📍 <b>ситуация:</b> " + plugin.EscapeHTML(plugin.FieldValue(fields, "situation")) + "\n")
	b.WriteString("💭 <b>мысли:</b> " + plugin.EscapeHTML(plugin.FieldValue(fields, "thoughts")) + "\n")
	b.WriteString("🎭 <b>эмоции:</b> " + plugin.EscapeHTML(plugin.FieldValue(fields, "emotions")))

	if resp := strings.TrimSpace(plugin.FieldValue(fields, "rational_response")); resp != "" {
		b.WriteString("\n⚖️ <b>рациональный ответ:</b> " + plugin.EscapeHTML(resp))
	}
	return b.String()
}

func (p *Emotions) FormatPreview(fields []model.EntryField) string {
	situation := plugin.FieldValue(fields, "situation")
	if situation == "" {
		return "мысли и эмоции"
	}
	return plugin.Truncate(situation, 20)
}
