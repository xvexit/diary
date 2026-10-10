package plugin

import (
	"context"
	"strings"

	"github.com/user/dnevnik-bot/internal/model"
)

// DiaryPlugin is the contract every diary type must implement.
// A plugin describes its metadata and the fields required to build an entry,
// validates raw answers, renders them both for the full view and the list,
// and owns its own storage.
type DiaryPlugin interface {
	Metadata() model.DiaryPluginMeta
	Fields() []model.FieldSpec
	Validate(fields []model.EntryField) error
	FormatEntry(fields []model.EntryField) string
	FormatPreview(fields []model.EntryField) string
	Store() EntryStore
}

// EntryStore is the per-diary storage contract. Each diary keeps its entries in
// its own table; a diary may replace this default implementation with its own
// typed storage later without touching the rest of the application.
type EntryStore interface {
	TableName() string
	Create(ctx context.Context, userID int64, content string, fields []model.EntryField) (*model.Entry, error)
	Get(ctx context.Context, userID int64, id int) (*model.Entry, error)
	List(ctx context.Context, userID int64, limit, offset int) ([]*model.Entry, error)
	Count(ctx context.Context, userID int64) (int, error)
	Update(ctx context.Context, userID int64, id int, content string, fields []model.EntryField) error
	Delete(ctx context.Context, userID int64, id int) error
	Search(ctx context.Context, userID int64, query string, limit, offset int) ([]*model.Entry, int, error)
	Random(ctx context.Context, userID int64) (*model.Entry, error)
}

// FieldValue returns the value of a named field, or an empty string.
func FieldValue(fields []model.EntryField, name string) string {
	for _, f := range fields {
		if f.Name == name {
			return f.Value
		}
	}
	return ""
}

// EscapeHTML escapes text the way Telegram's HTML parse mode expects.
func EscapeHTML(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
	).Replace(s)
}

// Truncate shortens s to max runes, appending an ellipsis when trimmed.
func Truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max]) + "..."
}
