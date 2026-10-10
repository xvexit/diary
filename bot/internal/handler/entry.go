package handler

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	tb "gopkg.in/telebot.v3"

	"github.com/user/dnevnik-bot/internal/model"
	"github.com/user/dnevnik-bot/internal/plugin"
	"github.com/user/dnevnik-bot/internal/repository"
	"github.com/user/dnevnik-bot/internal/service"
	"github.com/user/dnevnik-bot/internal/state"
)

const pageSize = 10
const catalogPageSize = 6

type EntryHandler struct {
	svc         *service.EntryService
	settingsSvc *service.SettingsService
	diarySvc    *service.DiaryService
	state       *state.Manager
	bot         *tb.Bot
}

func NewEntryHandler(
	svc *service.EntryService,
	settingsSvc *service.SettingsService,
	diarySvc *service.DiaryService,
	st *state.Manager,
	bot *tb.Bot,
) *EntryHandler {
	return &EntryHandler{
		svc:         svc,
		settingsSvc: settingsSvc,
		diarySvc:    diarySvc,
		state:       st,
		bot:         bot,
	}
}

func (h *EntryHandler) Register() {
	h.bot.Handle("/start", h.handleStart)
	h.bot.Handle(tb.OnText, h.handleText)

	h.bot.Handle(&tb.Btn{Unique: "new"}, h.answerAdapter(h.handleNewEntry))
	h.bot.Handle(&tb.Btn{Unique: "cancel"}, h.answerAdapter(h.handleCancel))
	h.bot.Handle(&tb.Btn{Unique: "menu"}, h.answerAdapter(h.handleMenu))
	h.bot.Handle(&tb.Btn{Unique: "noop"}, h.answerAdapter(h.handleNoop))
	h.bot.Handle(&tb.Btn{Unique: "random"}, h.answerAdapter(h.handleRandom))
	h.bot.Handle(&tb.Btn{Unique: "list"}, h.answerAdapter(h.handleList))
	h.bot.Handle(&tb.Btn{Unique: "view"}, h.answerAdapter(h.handleView))
	h.bot.Handle(&tb.Btn{Unique: "edit"}, h.answerAdapter(h.handleEdit))
	h.bot.Handle(&tb.Btn{Unique: "delete"}, h.answerAdapter(h.handleDeleteConfirm))
	h.bot.Handle(&tb.Btn{Unique: "delete_yes"}, h.answerAdapter(h.handleDeleteExec))
	h.bot.Handle(&tb.Btn{Unique: "search"}, h.answerAdapter(h.handleSearchResults))
	h.bot.Handle(&tb.Btn{Unique: "search_start"}, h.answerAdapter(h.handleSearch))
	h.bot.Handle(&tb.Btn{Unique: "settings"}, h.answerAdapter(h.handleSettings))
	h.bot.Handle(&tb.Btn{Unique: "toggle_reminder"}, h.answerAdapter(h.handleToggleReminder))
	h.bot.Handle(&tb.Btn{Unique: "change_time"}, h.answerAdapter(h.handleChangeTime))

	// diaries
	h.bot.Handle(&tb.Btn{Unique: "diaries"}, h.answerAdapter(h.handleDiaries))
	h.bot.Handle(&tb.Btn{Unique: "catalog"}, h.answerAdapter(h.handleCatalog))
	h.bot.Handle(&tb.Btn{Unique: "catalog_search"}, h.answerAdapter(h.handleCatalogSearch))
	h.bot.Handle(&tb.Btn{Unique: "d_install"}, h.answerAdapter(h.handleInstall))
	h.bot.Handle(&tb.Btn{Unique: "d_uninstall"}, h.answerAdapter(h.handleUninstall))
	h.bot.Handle(&tb.Btn{Unique: "d_open"}, h.answerAdapter(h.handleDiaryOpen))
	h.bot.Handle(&tb.Btn{Unique: "d_list"}, h.answerAdapter(h.handleDiaryList))
	h.bot.Handle(&tb.Btn{Unique: "d_new"}, h.answerAdapter(h.handleNewPluginEntry))
	h.bot.Handle(&tb.Btn{Unique: "p_choice"}, h.answerAdapter(h.handlePluginChoice))
	h.bot.Handle(&tb.Btn{Unique: "p_multi"}, h.answerAdapter(h.handlePluginMulti))
	h.bot.Handle(&tb.Btn{Unique: "p_done"}, h.answerAdapter(h.handlePluginMultiDone))
}

func (h *EntryHandler) answerAdapter(next func(c tb.Context) error) tb.HandlerFunc {
	return func(c tb.Context) error {
		return h.answer(c, next(c))
	}
}

func (h *EntryHandler) answer(c tb.Context, err error) error {
	if err != nil {
		c.Respond(&tb.CallbackResponse{Text: "❌ " + err.Error()})
	} else {
		c.Respond(&tb.CallbackResponse{})
	}
	return err
}

func (h *EntryHandler) handleNoop(c tb.Context) error { return nil }

// ── entry rendering helpers ───────────────────────

func (h *EntryHandler) entryFields(e *model.Entry) []model.EntryField {
	if len(e.Fields) > 0 {
		return e.Fields
	}
	return []model.EntryField{{Name: "text", Value: e.Content}}
}

func (h *EntryHandler) entryDisplay(slug string, e *model.Entry) string {
	if p, ok := h.diarySvc.Get(slug); ok {
		return p.FormatEntry(h.entryFields(e))
	}
	return plugin.EscapeHTML(e.Content)
}

func (h *EntryHandler) entryPreview(slug string, e *model.Entry) string {
	if p, ok := h.diarySvc.Get(slug); ok {
		return p.FormatPreview(h.entryFields(e))
	}
	return plugin.Truncate(e.Content, 20)
}

func (h *EntryHandler) menuMarkup() *tb.ReplyMarkup {
	m := &tb.ReplyMarkup{}
	m.Inline(
		m.Row(m.Data("📝 новая запись", "new"), m.Data("📋 мои записи", "list", "1")),
		m.Row(m.Data("📚 мои дневники", "diaries"), m.Data("🗂 база дневников", "catalog")),
		m.Row(m.Data("⚙️ настройки", "settings")),
	)
	return m
}

const menuText = "дневник\n\nличный дневник в telegram"

// ── date helpers ───────────────────────────────────

var weekdays = []string{"вс", "пн", "вт", "ср", "чт", "пт", "сб"}

// formatDate renders "02.01 (пн)".
func formatDate(t time.Time) string {
	return t.Format("02.01") + " (" + weekdays[int(t.Weekday())] + ")"
}

// formatDateTime renders "02.01.2006 · 15:04".
func formatDateTime(t time.Time) string {
	return t.Format("02.01.2006") + " · " + t.Format("15:04")
}

// ── Search ───────────────────────────────────────

func (h *EntryHandler) handleSearch(c tb.Context) error {
	uid := c.Sender().ID
	h.state.Set(uid, &state.UserState{State: state.Searching})

	markup := &tb.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ отмена", "cancel")))
	return c.Edit("🔍 <b>поиск</b>\n\nвведи слово или фразу для поиска по записям.", markup, tb.ModeHTML)
}

func (h *EntryHandler) performSearch(c tb.Context, query string) error {
	uid := c.Sender().ID
	h.state.Reset(uid)

	query = strings.TrimSpace(query)
	if query == "" {
		return c.Send("❌ введи что-нибудь для поиска.", tb.ModeHTML)
	}

	h.state.Set(uid, &state.UserState{State: state.Idle})

	msg, markup, err := h.buildSearchResults(uid, query, 1)
	if err != nil {
		return c.Send("❌ ошибка поиска.", tb.ModeHTML)
	}
	return c.Send(msg, markup, tb.ModeHTML)
}

func (h *EntryHandler) handleSearchResults(c tb.Context) error {
	uid := c.Sender().ID
	parts := strings.SplitN(c.Data(), "|", 2)
	query := parts[0]
	page := 1
	if len(parts) > 1 {
		page = parseInt(parts[1], 1)
	}

	msg, markup, err := h.buildSearchResults(uid, query, page)
	if err != nil {
		return c.Edit("❌ ошибка поиска.", tb.ModeHTML)
	}
	return c.Edit(msg, markup, tb.ModeHTML)
}

func (h *EntryHandler) buildSearchResults(uid int64, query string, page int) (string, *tb.ReplyMarkup, error) {
	slug := model.DefaultPluginSlug
	entries, total, err := h.svc.Search(context.Background(), uid, slug, query, page, pageSize)
	if err != nil {
		return "", nil, err
	}

	markup := &tb.ReplyMarkup{}

	if len(entries) == 0 {
		markup.Inline(
			markup.Row(markup.Data("🔍 повторить", "search_start"), markup.Data("🏠 в меню", "menu")),
		)
		return "поиск\n\nпо запросу «<i>" + plugin.EscapeHTML(query) + "</i>» ничего не найдено.", markup, nil
	}

	totalPages := (total + pageSize - 1) / pageSize
	msg := fmt.Sprintf("🔍 <b>поиск</b> · «%s»", plugin.EscapeHTML(query))

	var rows []tb.Row

	var navRow []tb.Btn
	if page > 1 {
		navRow = append(navRow, markup.Data("◀️", "search", query, strconv.Itoa(page-1)))
	}
	navRow = append(navRow, markup.Data(fmt.Sprintf("%d/%d", page, totalPages), "noop"))
	if page < totalPages {
		navRow = append(navRow, markup.Data("▶️", "search", query, strconv.Itoa(page+1)))
	}
	rows = append(rows, navRow)

	for _, e := range entries {
		label := h.entryPreview(slug, e) + " 📅 " + e.CreatedAt.Format("02.01")
		rows = append(rows, markup.Row(markup.Data(label, "view", slug, strconv.Itoa(e.ID))))
	}
	rows = append(rows, markup.Row(markup.Data("🔍 повторить", "search_start"), markup.Data("🏠 в меню", "menu")))

	markup.Inline(rows...)
	return msg, markup, nil
}

// ── Diaries ───────────────────────────────────────

func (h *EntryHandler) handleDiaries(c tb.Context) error {
	uid := c.Sender().ID
	h.state.Reset(uid)

	added, err := h.diarySvc.Added(context.Background(), uid)
	if err != nil {
		return c.Edit("❌ ошибка загрузки дневников.", tb.ModeHTML)
	}

	m := &tb.ReplyMarkup{}
	var rows []tb.Row
	for _, p := range added {
		meta := p.Metadata()
		rows = append(rows, m.Row(m.Data(meta.Icon+" "+meta.Name, "d_open", meta.Slug)))
	}
	rows = append(rows, m.Row(m.Data("🗂 база дневников", "catalog"), m.Data("🏠 в меню", "menu")))
	m.Inline(rows...)

	header := "📚 <b>мои дневники</b>"
	if len(added) == 0 {
		header += "\n\nу тебя пока нет дневников. загляни в базу и добавь."
	}
	return c.Edit(header, m, tb.ModeHTML)
}

func (h *EntryHandler) handleCatalog(c tb.Context) error {
	uid := c.Sender().ID
	data := strings.TrimSpace(c.Data())
	page := 1
	if data == "" {
		h.state.Reset(uid)
	} else {
		page = parseInt(data, 1)
	}
	return h.renderCatalog(c, uid, page, true)
}

// renderCatalog draws the diary catalogue (без описаний) with name search and
// pagination. The active query and page are kept in the user's state so
// install/uninstall can re-render the same view.
func (h *EntryHandler) renderCatalog(c tb.Context, uid int64, page int, edit bool) error {
	items, err := h.diarySvc.Catalog(context.Background(), uid)
	if err != nil {
		if edit {
			return c.Edit("ошибка загрузки базы дневников.", tb.ModeHTML)
		}
		return c.Send("ошибка загрузки базы дневников.", tb.ModeHTML)
	}

	query := strings.TrimSpace(h.state.Get(uid).CatalogQuery)
	if query != "" {
		q := strings.ToLower(query)
		filtered := make([]service.CatalogItem, 0, len(items))
		for _, it := range items {
			if strings.Contains(strings.ToLower(it.Diary.Metadata().Name), q) {
				filtered = append(filtered, it)
			}
		}
		items = filtered
	}

	total := len(items)
	totalPages := (total + catalogPageSize - 1) / catalogPageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}
	start := (page - 1) * catalogPageSize
	end := start + catalogPageSize
	if end > total {
		end = total
	}
	pageItems := items[start:end]

	us := h.state.Get(uid)
	us.CatalogQuery = query
	us.CatalogPage = page
	h.state.Set(uid, us)

	var text strings.Builder
	text.WriteString("<b>база дневников</b>")
	if query != "" {
		fmt.Fprintf(&text, "\n\nпоиск: «%s»", plugin.EscapeHTML(query))
	}

	m := &tb.ReplyMarkup{}
	var rows []tb.Row

	for _, it := range pageItems {
		meta := it.Diary.Metadata()
		if it.Installed {
			rows = append(rows, m.Row(m.Data("✅ "+meta.Icon+" "+meta.Name, "d_uninstall", meta.Slug)))
		} else {
			rows = append(rows, m.Row(m.Data("➕ "+meta.Icon+" "+meta.Name, "d_install", meta.Slug)))
		}
	}

	if totalPages > 1 {
		var navRow []tb.Btn
		if page > 1 {
			navRow = append(navRow, m.Data("◀️", "catalog", strconv.Itoa(page-1)))
		}
		navRow = append(navRow, m.Data(fmt.Sprintf("%d/%d", page, totalPages), "noop"))
		if page < totalPages {
			navRow = append(navRow, m.Data("▶️", "catalog", strconv.Itoa(page+1)))
		}
		rows = append(rows, navRow)
	}

	navButtons := []tb.Btn{m.Data("🔍 поиск", "catalog_search")}
	if query != "" {
		navButtons = append(navButtons, m.Data("✖️ сбросить", "catalog"))
	}
	navButtons = append(navButtons, m.Data("📚 мои дневники", "diaries"), m.Data("🏠 в меню", "menu"))
	rows = append(rows, navButtons)
	m.Inline(rows...)

	return h.reply(c, edit, text.String(), m)
}

func (h *EntryHandler) handleCatalogSearch(c tb.Context) error {
	uid := c.Sender().ID
	us := h.state.Get(uid)
	us.State = state.CatalogSearching
	h.state.Set(uid, us)

	markup := &tb.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ отмена", "catalog")))
	return c.Edit("🔍 <b>поиск</b>\n\nвведи название дневника.", markup, tb.ModeHTML)
}

func (h *EntryHandler) handleInstall(c tb.Context) error {
	uid := c.Sender().ID
	if err := h.diarySvc.Install(context.Background(), uid, c.Data()); err != nil {
		return err
	}
	return h.renderCatalog(c, uid, h.state.Get(uid).CatalogPage, true)
}

func (h *EntryHandler) handleUninstall(c tb.Context) error {
	uid := c.Sender().ID
	if err := h.diarySvc.Uninstall(context.Background(), uid, c.Data()); err != nil {
		return err
	}
	return h.renderCatalog(c, uid, h.state.Get(uid).CatalogPage, true)
}

func (h *EntryHandler) handleDiaryOpen(c tb.Context) error {
	uid := c.Sender().ID
	h.state.Reset(uid)
	return h.buildDiaryList(c, uid, c.Data(), 1, true)
}

func (h *EntryHandler) handleDiaryList(c tb.Context) error {
	uid := c.Sender().ID
	parts := strings.SplitN(c.Data(), "|", 2)
	slug := parts[0]
	page := 1
	if len(parts) > 1 {
		page = parseInt(parts[1], 1)
	}
	return h.buildDiaryList(c, uid, slug, page, true)
}

// buildDiaryList renders the entries of one diary that is not the default one.
func (h *EntryHandler) buildDiaryList(c tb.Context, uid int64, slug string, page int, edit bool) error {
	p, ok := h.diarySvc.Get(slug)
	if !ok {
		return c.Edit("❌ неизвестный дневник.", tb.ModeHTML)
	}
	meta := p.Metadata()

	entries, total, err := h.svc.List(context.Background(), uid, slug, page, pageSize)
	if err != nil {
		return c.Edit("❌ ошибка загрузки записей.", tb.ModeHTML)
	}

	m := &tb.ReplyMarkup{}

	if total == 0 {
		m.Inline(
			m.Row(m.Data("📝 новая запись", "d_new", slug)),
			m.Row(m.Data("🏠 в меню", "menu")),
		)
		msg := fmt.Sprintf("%s <b>%s</b>\n\nпока нет записей.", meta.Icon, plugin.EscapeHTML(meta.Name))
		return h.reply(c, edit, msg, m)
	}

	totalPages := (total + pageSize - 1) / pageSize
	msg := fmt.Sprintf("%s <b>%s</b>", meta.Icon, plugin.EscapeHTML(meta.Name))

	var rows []tb.Row

	var navRow []tb.Btn
	if page > 1 {
		navRow = append(navRow, m.Data("◀️", "d_list", slug, strconv.Itoa(page-1)))
	}
	navRow = append(navRow, m.Data(fmt.Sprintf("%d/%d", page, totalPages), "noop"))
	if page < totalPages {
		navRow = append(navRow, m.Data("▶️", "d_list", slug, strconv.Itoa(page+1)))
	}
	rows = append(rows, navRow)

	for _, e := range entries {
		label := h.entryPreview(slug, e) + " 📅 " + e.CreatedAt.Format("02.01")
		rows = append(rows, m.Row(m.Data(label, "view", slug, strconv.Itoa(e.ID))))
	}
	rows = append(rows, m.Row(m.Data("📝 новая запись", "d_new", slug), m.Data("🏠 в меню", "menu")))
	m.Inline(rows...)

	return h.reply(c, edit, msg, m)
}

func (h *EntryHandler) reply(c tb.Context, edit bool, text string, markup *tb.ReplyMarkup) error {
	if edit {
		return c.Edit(text, markup, tb.ModeHTML)
	}
	return c.Send(text, markup, tb.ModeHTML)
}

// ── Plugin entry FSM ──────────────────────────────

func (h *EntryHandler) handleNewPluginEntry(c tb.Context) error {
	uid := c.Sender().ID
	slug := c.Data()

	if _, ok := h.diarySvc.Get(slug); !ok {
		return c.Edit("❌ неизвестный дневник.", tb.ModeHTML)
	}

	h.state.Set(uid, &state.UserState{
		State:      state.CreatingPluginEntry,
		PluginSlug: slug,
		FieldIndex: 0,
		Values:     make(map[string]string),
	})

	return h.renderCurrentField(c, uid, true)
}

func (h *EntryHandler) renderCurrentField(c tb.Context, uid int64, edit bool) error {
	us := h.state.Get(uid)
	p, ok := h.diarySvc.Get(us.PluginSlug)
	if !ok || us.FieldIndex >= len(p.Fields()) {
		h.state.Reset(uid)
		return c.Send("❌ не удалось продолжить создание записи.", tb.ModeHTML)
	}

	spec := p.Fields()[us.FieldIndex]
	text, markup := h.fieldPrompt(us, spec)
	return h.reply(c, edit, text, markup)
}

func (h *EntryHandler) fieldPrompt(us *state.UserState, spec model.FieldSpec) (string, *tb.ReplyMarkup) {
	total := 0
	header := ""
	if p, ok := h.diarySvc.Get(us.PluginSlug); ok {
		meta := p.Metadata()
		total = len(p.Fields())
		header = fmt.Sprintf("%s <b>%s</b>\n\n%s", meta.Icon, plugin.EscapeHTML(meta.Name), plugin.EscapeHTML(spec.Prompt))
	} else {
		header = plugin.EscapeHTML(spec.Prompt)
	}
	header += fmt.Sprintf("\n\n<i>шаг %d из %d</i>", us.FieldIndex+1, total)

	m := &tb.ReplyMarkup{}
	var rows []tb.Row

	switch spec.Type {
	case model.FieldTypeChoice:
		if spec.Multiple {
			for _, opt := range spec.Options {
				mark := "⬜"
				for _, s := range us.Selected {
					if s == opt {
						mark = "☑️"
						break
					}
				}
				rows = append(rows, m.Row(m.Data(mark+" "+opt, "p_multi", opt)))
			}
			rows = append(rows, m.Row(m.Data("✅ готово", "p_done")))
		} else {
			for _, opt := range spec.Options {
				rows = append(rows, m.Row(m.Data(opt, "p_choice", spec.Name, opt)))
			}
		}
	case model.FieldTypeScale:
		var cur []tb.Btn
		for n := spec.Min; n <= spec.Max; n++ {
			cur = append(cur, m.Data(strconv.Itoa(n), "p_choice", spec.Name, strconv.Itoa(n)))
			if len(cur) == 5 {
				rows = append(rows, m.Row(cur...))
				cur = nil
			}
		}
		if len(cur) > 0 {
			rows = append(rows, m.Row(cur...))
		}
	}

	rows = append(rows, m.Row(m.Data("❌ отмена", "cancel")))
	m.Inline(rows...)
	return header, m
}

func (h *EntryHandler) advancePluginEntry(c tb.Context, uid int64, edit bool) error {
	us := h.state.Get(uid)
	p, ok := h.diarySvc.Get(us.PluginSlug)
	if !ok {
		h.state.Reset(uid)
		return c.Send("❌ неизвестный дневник.", tb.ModeHTML)
	}

	if us.FieldIndex+1 >= len(p.Fields()) {
		return h.finalizePluginEntry(c, uid, p)
	}

	us.FieldIndex++
	us.Selected = nil
	h.state.Set(uid, us)
	return h.renderCurrentField(c, uid, edit)
}

func (h *EntryHandler) finalizePluginEntry(c tb.Context, uid int64, p plugin.DiaryPlugin) error {
	us := h.state.Get(uid)

	specs := p.Fields()
	built := make([]model.EntryField, 0, len(specs))
	for _, spec := range specs {
		built = append(built, model.EntryField{Name: spec.Name, Value: us.Values[spec.Name]})
	}

	slug := us.PluginSlug
	entry, err := h.svc.Create(context.Background(), uid, slug, built)
	if err != nil {
		h.state.Reset(uid)
		m := &tb.ReplyMarkup{}
		m.Inline(m.Row(m.Data("🏠 в меню", "menu")))
		return c.Send("❌ "+err.Error(), m, tb.ModeHTML)
	}

	h.state.Reset(uid)

	m := &tb.ReplyMarkup{}
	m.Inline(m.Row(
		m.Data("📚 к дневнику", "d_open", slug),
		m.Data("🏠 в меню", "menu"),
	))
	return c.Send(fmt.Sprintf("✅ запись #%d сохранена!", entry.ID), m, tb.ModeHTML)
}

func (h *EntryHandler) handlePluginChoice(c tb.Context) error {
	uid := c.Sender().ID
	us := h.state.Get(uid)
	if us.State != state.CreatingPluginEntry {
		return nil
	}

	parts := strings.SplitN(c.Data(), "|", 2)
	if len(parts) != 2 {
		return nil
	}
	name, val := parts[0], parts[1]

	p, ok := h.diarySvc.Get(us.PluginSlug)
	if !ok || us.FieldIndex >= len(p.Fields()) {
		return nil
	}
	spec := p.Fields()[us.FieldIndex]
	if spec.Name != name {
		return nil
	}

	if us.Values == nil {
		us.Values = make(map[string]string)
	}
	us.Values[spec.Name] = val
	return h.advancePluginEntry(c, uid, true)
}

func (h *EntryHandler) handlePluginMulti(c tb.Context) error {
	uid := c.Sender().ID
	us := h.state.Get(uid)
	if us.State != state.CreatingPluginEntry {
		return nil
	}

	val := c.Data()
	found := false
	for i, s := range us.Selected {
		if s == val {
			us.Selected = append(us.Selected[:i], us.Selected[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		us.Selected = append(us.Selected, val)
	}
	h.state.Set(uid, us)
	return h.renderCurrentField(c, uid, true)
}

func (h *EntryHandler) handlePluginMultiDone(c tb.Context) error {
	uid := c.Sender().ID
	us := h.state.Get(uid)
	if us.State != state.CreatingPluginEntry {
		return nil
	}

	p, ok := h.diarySvc.Get(us.PluginSlug)
	if !ok || us.FieldIndex >= len(p.Fields()) {
		return nil
	}
	spec := p.Fields()[us.FieldIndex]

	if spec.Required && len(us.Selected) == 0 {
		return errors.New("выбери хотя бы один вариант")
	}

	if us.Values == nil {
		us.Values = make(map[string]string)
	}
	us.Values[spec.Name] = strings.Join(us.Selected, ", ")
	us.Selected = nil
	return h.advancePluginEntry(c, uid, true)
}

// ── Settings ──────────────────────────────────────

func (h *EntryHandler) handleSettings(c tb.Context) error {
	uid := c.Sender().ID
	h.state.Reset(uid)

	settings, err := h.settingsSvc.Get(context.Background(), uid)
	if err != nil {
		_ = h.settingsSvc.Upsert(context.Background(), uid)
		settings = &repository.UserSettings{
			UserID:          uid,
			ReminderEnabled: true,
			ReminderTime:    "21:00",
		}
	}

	status := "✅ включены"
	if !settings.ReminderEnabled {
		status = "❌ выключены"
	}

	markup := &tb.ReplyMarkup{}
	markup.Inline(
		markup.Row(markup.Data("🔄 вкл/выкл", "toggle_reminder")),
		markup.Row(markup.Data("⏰ сменить время", "change_time")),
		markup.Row(markup.Data("🏠 в меню", "menu")),
	)

	msg := fmt.Sprintf(
		"⚙️ <b>настройки</b>\n\n📅 <b>напоминание:</b> %s\n⏰ <b>время:</b> <i>%s</i>",
		status, plugin.EscapeHTML(settings.ReminderTime),
	)

	return c.Edit(msg, markup, tb.ModeHTML)
}

func (h *EntryHandler) handleToggleReminder(c tb.Context) error {
	uid := c.Sender().ID
	if err := h.settingsSvc.ToggleReminder(context.Background(), uid); err != nil {
		return c.Edit("❌ ошибка.", tb.ModeHTML)
	}
	return h.handleSettings(c)
}

func (h *EntryHandler) handleChangeTime(c tb.Context) error {
	uid := c.Sender().ID
	h.state.Set(uid, &state.UserState{State: state.ChangingTime})

	markup := &tb.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ отмена", "cancel")))
	return c.Edit("⏰ <b>новое время</b>\n\nвведи время в формате <b>HH:MM</b> (например 21:00):", markup, tb.ModeHTML)
}

func parseInt(data string, defaultVal int) int {
	n, err := strconv.Atoi(data)
	if err != nil || n < 1 {
		return defaultVal
	}
	return n
}

// ── Menu / entry CRUD ─────────────────────────────

func (h *EntryHandler) handleStart(c tb.Context) error {
	h.state.Reset(c.Sender().ID)
	return c.Send(menuText, h.menuMarkup(), tb.ModeHTML)
}

func (h *EntryHandler) handleText(c tb.Context) error {
	uid := c.Sender().ID
	us := h.state.Get(uid)

	switch us.State {
	case state.Creating:
		content := strings.TrimSpace(c.Text())
		if content == "" {
			return c.Send("❌ запись не может быть пустой.", tb.ModeHTML)
		}
		fields := []model.EntryField{{Name: "text", Value: content}}
		entry, err := h.svc.Create(context.Background(), uid, model.DefaultPluginSlug, fields)
		if err != nil {
			return c.Send("❌ ошибка при сохранении: "+err.Error(), tb.ModeHTML)
		}
		h.state.Reset(uid)

		markup := &tb.ReplyMarkup{}
		markup.Inline(
			markup.Row(markup.Data("📋 к списку", "list", "1"), markup.Data("🏠 в меню", "menu")),
		)

		return c.Send(fmt.Sprintf("✅ запись #%d сохранена!", entry.ID), markup, tb.ModeHTML)

	case state.Editing:
		content := strings.TrimSpace(c.Text())
		if content == "" {
			return c.Send("❌ запись не может быть пустой.", tb.ModeHTML)
		}
		fields := []model.EntryField{{Name: "text", Value: content}}
		if err := h.svc.Update(context.Background(), uid, model.DefaultPluginSlug, us.EditEntryID, fields); err != nil {
			return c.Send("❌ ошибка при обновлении: "+err.Error(), tb.ModeHTML)
		}
		h.state.Reset(uid)

		markup := &tb.ReplyMarkup{}
		markup.Inline(
			markup.Row(markup.Data("📋 к списку", "list", "1"), markup.Data("🏠 в меню", "menu")),
		)

		return c.Send("✅ запись обновлена!", markup, tb.ModeHTML)

	case state.Searching:
		return h.performSearch(c, c.Text())

	case state.CatalogSearching:
		us := h.state.Get(uid)
		us.State = state.Idle
		us.CatalogQuery = strings.TrimSpace(c.Text())
		h.state.Set(uid, us)
		return h.renderCatalog(c, uid, 1, false)

	case state.ChangingTime:
		t := strings.TrimSpace(c.Text())
		if err := h.settingsSvc.SetReminderTime(context.Background(), uid, t); err != nil {
			return c.Send("❌ "+err.Error(), tb.ModeHTML)
		}
		h.state.Reset(uid)

		markup := &tb.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("⚙️ настройки", "settings"), markup.Data("🏠 в меню", "menu")))
		return c.Send("✅ время напоминания установлено на <b>"+plugin.EscapeHTML(t)+"</b>", markup, tb.ModeHTML)

	case state.CreatingPluginEntry:
		return h.handlePluginText(c, uid, us)

	default:
		return c.Send("❌ непонятная команда. используй /start.", tb.ModeHTML)
	}
}

func (h *EntryHandler) handlePluginText(c tb.Context, uid int64, us *state.UserState) error {
	p, ok := h.diarySvc.Get(us.PluginSlug)
	if !ok || us.FieldIndex >= len(p.Fields()) {
		h.state.Reset(uid)
		return c.Send("❌ не удалось продолжить создание записи.", tb.ModeHTML)
	}

	spec := p.Fields()[us.FieldIndex]
	if spec.Type != model.FieldTypeText {
		return c.Send("❌ для этого шага используй кнопки ниже.", tb.ModeHTML)
	}

	val := strings.TrimSpace(c.Text())
	if spec.Required && val == "" {
		return c.Send("❌ это поле обязательно.", tb.ModeHTML)
	}

	if us.Values == nil {
		us.Values = make(map[string]string)
	}
	us.Values[spec.Name] = val
	return h.advancePluginEntry(c, uid, false)
}

func (h *EntryHandler) handleNewEntry(c tb.Context) error {
	uid := c.Sender().ID
	h.state.Set(uid, &state.UserState{State: state.Creating})

	markup := &tb.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ отмена", "cancel")))

	return c.Edit("✍️ <b>напиши свою запись:</b>", markup, tb.ModeHTML)
}

func (h *EntryHandler) handleCancel(c tb.Context) error {
	uid := c.Sender().ID
	h.state.Reset(uid)

	markup := &tb.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🏠 в меню", "menu")))

	return c.Edit("❌ отменено.", markup, tb.ModeHTML)
}

func (h *EntryHandler) handleMenu(c tb.Context) error {
	h.state.Reset(c.Sender().ID)
	return c.Edit(menuText, h.menuMarkup(), tb.ModeHTML)
}

// handleList renders the default (free-form) diary list with search and random.
func (h *EntryHandler) handleList(c tb.Context) error {
	uid := c.Sender().ID
	page := parseInt(c.Data(), 1)
	slug := model.DefaultPluginSlug

	entries, total, err := h.svc.List(context.Background(), uid, slug, page, pageSize)
	if err != nil {
		return c.Edit("❌ ошибка загрузки записей.", tb.ModeHTML)
	}

	markup := &tb.ReplyMarkup{}

	if total == 0 {
		markup.Inline(
			markup.Row(markup.Data("📝 новая запись", "new")),
			markup.Row(markup.Data("🏠 в меню", "menu")),
		)
		return c.Edit("📋 <b>мои записи</b>\n\nу тебя пока нет записей. напиши первую! ✍️", markup, tb.ModeHTML)
	}

	totalPages := (total + pageSize - 1) / pageSize
	msg := fmt.Sprintf("📋 <b>мои записи</b>  %d/%d", page, totalPages)

	var rows []tb.Row

	var navRow []tb.Btn
	if page > 1 {
		navRow = append(navRow, markup.Data("◀️", "list", strconv.Itoa(page-1)))
	}
	navRow = append(navRow, markup.Data(fmt.Sprintf("%d/%d", page, totalPages), "noop"))
	if page < totalPages {
		navRow = append(navRow, markup.Data("▶️", "list", strconv.Itoa(page+1)))
	}
	rows = append(rows, navRow)

	for _, e := range entries {
		label := h.entryPreview(slug, e) + " 📅 " + e.CreatedAt.Format("02.01")
		rows = append(rows, markup.Row(markup.Data(label, "view", slug, strconv.Itoa(e.ID))))
	}

	rows = append(rows,
		markup.Row(markup.Data("📝 новая запись", "new"), markup.Data("🔍 поиск", "search_start")),
		markup.Row(markup.Data("🎲 сюрприз", "random"), markup.Data("🏠 в меню", "menu")),
	)

	markup.Inline(rows...)
	return c.Edit(msg, markup, tb.ModeHTML)
}

func (h *EntryHandler) handleView(c tb.Context) error {
	uid := c.Sender().ID
	parts := strings.SplitN(c.Data(), "|", 2)
	if len(parts) != 2 {
		return c.Edit("❌ запись не найдена.", tb.ModeHTML)
	}
	slug := parts[0]
	id, _ := strconv.Atoi(parts[1])

	entry, err := h.svc.GetByID(context.Background(), uid, slug, id)
	if err != nil {
		return c.Edit("❌ запись не найдена.", tb.ModeHTML)
	}

	var msg string
	if entry.CreatedAt.Equal(entry.UpdatedAt) {
		msg = fmt.Sprintf("<b>#%d</b>\n\n%s\n\n📅 <i>создана: %s</i>",
			entry.ID, h.entryDisplay(slug, entry),
			entry.CreatedAt.Format("02.01.2006 15:04"))
	} else {
		msg = fmt.Sprintf("<b>#%d</b>\n\n%s\n\n📅 <i>создана: %s</i>\n🔄 <i>изменена: %s</i>",
			entry.ID, h.entryDisplay(slug, entry),
			entry.CreatedAt.Format("02.01.2006 15:04"),
			entry.UpdatedAt.Format("02.01.2006 15:04"))
	}

	markup := &tb.ReplyMarkup{}
	if slug == model.DefaultPluginSlug {
		markup.Inline(
			markup.Row(
				markup.Data("✏️ редактировать", "edit", strconv.Itoa(entry.ID)),
				markup.Data("🗑 удалить", "delete", slug, strconv.Itoa(entry.ID)),
			),
			markup.Row(
				markup.Data("⬅️ назад", "list", "1"),
				markup.Data("🏠 в меню", "menu"),
			),
		)
	} else {
		markup.Inline(
			markup.Row(markup.Data("🗑 удалить", "delete", slug, strconv.Itoa(entry.ID))),
			markup.Row(
				markup.Data("📚 к дневнику", "d_open", slug),
				markup.Data("🏠 в меню", "menu"),
			),
		)
	}

	return c.Edit(msg, markup, tb.ModeHTML)
}

func (h *EntryHandler) handleEdit(c tb.Context) error {
	uid := c.Sender().ID
	id, _ := strconv.Atoi(c.Data())

	entry, err := h.svc.GetByID(context.Background(), uid, model.DefaultPluginSlug, id)
	if err != nil {
		return c.Edit("❌ запись не найдена.", tb.ModeHTML)
	}

	h.state.Set(uid, &state.UserState{State: state.Editing, EditEntryID: id})

	markup := &tb.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ отмена", "cancel")))

	msg := fmt.Sprintf("✏️ <b>редактирование #%d</b>\n\n<code>%s</code>\n\n<i>для редактирования записи нажми на текст выше вставь в поле ввода и отправляй</i>",
		entry.ID, plugin.EscapeHTML(entry.Content))

	return c.Edit(msg, markup, tb.ModeHTML)
}

func (h *EntryHandler) handleDeleteConfirm(c tb.Context) error {
	parts := strings.SplitN(c.Data(), "|", 2)
	if len(parts) != 2 {
		return c.Edit("❌ запись не найдена.", tb.ModeHTML)
	}
	slug, id := parts[0], parts[1]

	backUnique, backData := "list", "1"
	if slug != model.DefaultPluginSlug {
		backUnique, backData = "d_open", slug
	}

	markup := &tb.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.Data("✅ да, удалить", "delete_yes", slug, id),
			markup.Data("❌ нет", backUnique, backData),
		),
	)

	return c.Edit("🗑 <b>точно удалить?</b>", markup, tb.ModeHTML)
}

func (h *EntryHandler) handleDeleteExec(c tb.Context) error {
	uid := c.Sender().ID
	parts := strings.SplitN(c.Data(), "|", 2)
	if len(parts) != 2 {
		return c.Edit("❌ ошибка при удалении.", tb.ModeHTML)
	}
	slug := parts[0]
	id, _ := strconv.Atoi(parts[1])

	if err := h.svc.Delete(context.Background(), uid, slug, id); err != nil {
		return c.Edit("❌ ошибка при удалении: "+err.Error(), tb.ModeHTML)
	}

	backUnique, backData := "list", "1"
	backLabel := "📋 к списку"
	if slug != model.DefaultPluginSlug {
		backUnique, backData = "d_open", slug
		backLabel = "📚 к дневнику"
	}

	markup := &tb.ReplyMarkup{}
	markup.Inline(
		markup.Row(markup.Data(backLabel, backUnique, backData), markup.Data("🏠 в меню", "menu")),
	)

	return c.Edit("🗑 запись удалена.", markup, tb.ModeHTML)
}

func (h *EntryHandler) handleRandom(c tb.Context) error {
	uid := c.Sender().ID
	slug := model.DefaultPluginSlug

	entry, err := h.svc.Random(context.Background(), uid, slug)
	if err != nil {
		return c.Edit("❌ у тебя пока нет записей.", tb.ModeHTML)
	}

	msg := fmt.Sprintf("🎲 <b>случайная запись #%d</b>\n\n%s\n\n📅 <i>%s</i>",
		entry.ID, h.entryDisplay(slug, entry), entry.CreatedAt.Format("02.01.2006 15:04"))

	markup := &tb.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.Data("👁 просмотр", "view", slug, strconv.Itoa(entry.ID)),
			markup.Data("🎲 ещё", "random"),
			markup.Data("🏠 в меню", "menu"),
		),
	)

	return c.Edit(msg, markup, tb.ModeHTML)
}
