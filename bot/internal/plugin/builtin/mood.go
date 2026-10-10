package builtin

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/user/dnevnik-bot/internal/model"
	"github.com/user/dnevnik-bot/internal/plugin"
)

const SlugMood = "mood"

// Mood is the built-in mood tracker.
type Mood struct {
	store plugin.EntryStore
}

func NewMood(store plugin.EntryStore) *Mood { return &Mood{store: store} }

func (p *Mood) Store() plugin.EntryStore { return p.store }

func (p *Mood) Metadata() model.DiaryPluginMeta {
	return model.DiaryPluginMeta{
		Slug:        SlugMood,
		Name:        "дневник настроения",
		Description: "отмечай настроение, уровень энергии и заметки.",
		Icon:        "🌤",
	}
}

func (p *Mood) Fields() []model.FieldSpec {
	return []model.FieldSpec{
		{
			Name:     "mood",
			Prompt:   "оцени настроение (1–10)",
			Type:     model.FieldTypeScale,
			Min:      1,
			Max:      10,
			Required: true,
		},
		{
			Name:     "energy",
			Prompt:   "уровень энергии (1–10)",
			Type:     model.FieldTypeScale,
			Min:      1,
			Max:      10,
			Required: true,
		},
		{
			Name:     "note",
			Prompt:   "заметка (что повлияло на настроение?)",
			Type:     model.FieldTypeText,
			Required: false,
		},
	}
}

func (p *Mood) Validate(fields []model.EntryField) error {
	mood := strings.TrimSpace(plugin.FieldValue(fields, "mood"))
	if mood == "" {
		return errors.New("выберите настроение")
	}
	energy := strings.TrimSpace(plugin.FieldValue(fields, "energy"))
	n, err := strconv.Atoi(energy)
	if err != nil || n < 1 || n > 10 {
		return errors.New("энергия должна быть числом от 1 до 10")
	}
	return nil
}

func (p *Mood) FormatEntry(fields []model.EntryField) string {
	var b strings.Builder
	b.WriteString("<b>🌤 дневник настроения</b>\n\n")
	fmt.Fprintf(&b, "настроение: <b>%s</b>\n", plugin.EscapeHTML(plugin.FieldValue(fields, "mood")))
	fmt.Fprintf(&b, "энергия: <b>%s</b>/10", plugin.EscapeHTML(plugin.FieldValue(fields, "energy")))

	if note := strings.TrimSpace(plugin.FieldValue(fields, "note")); note != "" {
		fmt.Fprintf(&b, "\n\n📝 %s", plugin.EscapeHTML(note))
	}
	return b.String()
}

func (p *Mood) FormatPreview(fields []model.EntryField) string {
	mood := plugin.FieldValue(fields, "mood")
	energy := plugin.FieldValue(fields, "energy")
	return fmt.Sprintf("%s · энергия %s", mood, energy)
}
