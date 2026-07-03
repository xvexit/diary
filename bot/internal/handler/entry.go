package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tb "gopkg.in/telebot.v3"

	"github.com/user/dnevnik-bot/internal/repository"
	"github.com/user/dnevnik-bot/internal/service"
	"github.com/user/dnevnik-bot/internal/state"
)

const pageSize = 10

type EntryHandler struct {
	svc         *service.EntryService
	settingsSvc *service.SettingsService
	state       *state.Manager
	bot         *tb.Bot
}

func NewEntryHandler(svc *service.EntryService, settingsSvc *service.SettingsService, st *state.Manager, bot *tb.Bot) *EntryHandler {
	return &EntryHandler{
		svc:         svc,
		settingsSvc: settingsSvc,
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
}

func (h *EntryHandler) answerAdapter(next func(c tb.Context) error) tb.HandlerFunc {
	return func(c tb.Context) error {
		return h.answer(c, next(c))
	}
}

// ── Search ───────────────────────────────────────

func (h *EntryHandler) handleSearch(c tb.Context) error {
	uid := c.Sender().ID
	h.state.Set(uid, &state.UserState{State: state.Searching})

	markup := &tb.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ отмена", "cancel")))
	return c.Edit("🔍 <b>поиск</b>\n\nвведи слово или фразу для поиска по записям:", markup, tb.ModeHTML)
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
	data := c.Data()
	parts := strings.SplitN(data, "|", 2)
	query := parts[0]
	page := 1
	if len(parts) > 1 {
		page, _ = strconv.Atoi(parts[1])
		if page < 1 {
			page = 1
		}
	}

	msg, markup, err := h.buildSearchResults(uid, query, page)
	if err != nil {
		return c.Edit("❌ ошибка поиска.", tb.ModeHTML)
	}
	return c.Edit(msg, markup, tb.ModeHTML)
}

func (h *EntryHandler) buildSearchResults(uid int64, query string, page int) (string, *tb.ReplyMarkup, error) {
	entries, total, err := h.svc.Search(context.Background(), uid, query, page, pageSize)
	if err != nil {
		return "", nil, err
	}

	markup := &tb.ReplyMarkup{}

	if len(entries) == 0 {
		markup.Inline(
			markup.Row(markup.Data("🔍 снова", "search_start"), markup.Data("🏠 в меню", "menu")),
		)
		return "🔍 <b>поиск</b>\n\nничего не найдено по запросу «<i>" + escapeHTML(query) + "</i>».", markup, nil
	}

	totalPages := (total + pageSize - 1) / pageSize
	msg := fmt.Sprintf("🔍 <b>поиск: «%s»</b>  %d/%d", escapeHTML(query), page, totalPages)

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
		label := firstWords(e.Content, 20) + " 📅 " + e.CreatedAt.Format("02.01")
		rows = append(rows, markup.Row(markup.Data(label, "view", strconv.Itoa(e.ID))))
	}
	rows = append(rows, markup.Row(markup.Data("🔍 снова", "search_start"), markup.Data("🏠 в меню", "menu")))

	markup.Inline(rows...)
	return msg, markup, nil
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
		markup.Row(markup.Data("🔄 Вкл/Выкл", "toggle_reminder")),
		markup.Row(markup.Data("⏰ сменить время", "change_time")),
		markup.Row(markup.Data("🏠 в меню", "menu")),
	)

	msg := fmt.Sprintf(
		"⚙️ <b>настройки</b>\n\n📅 <b>напоминание:</b> %s\n⏰ <b>время:</b> <i>%s</i>",
		status, escapeHTML(settings.ReminderTime),
	)

	return c.Edit(msg, markup, tb.ModeHTML)
}

func (h *EntryHandler) handleToggleReminder(c tb.Context) error {
	uid := c.Sender().ID
	err := h.settingsSvc.ToggleReminder(context.Background(), uid)
	if err != nil {
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

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func (h *EntryHandler) handleStart(c tb.Context) error {
	h.state.Reset(c.Sender().ID)

	msg := "📔 <b>дневник</b>\n\nтвой личный дневник в telegram. пиши записи, ищи их и получай напоминания."

	markup := &tb.ReplyMarkup{}
	markup.Inline(
		markup.Row(markup.Data("📝 новая запись", "new"), markup.Data("🎲 сюрприз", "random")),
		markup.Row(markup.Data("📋 мои записи", "list", "1"), markup.Data("⚙️ настройки", "settings")),
	)

	return c.Send(msg, markup, tb.ModeHTML)
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
		entry, err := h.svc.Create(context.Background(), uid, content)
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
		err := h.svc.Update(context.Background(), uid, us.EditEntryID, content)
		if err != nil {
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

	case state.ChangingTime:
		t := strings.TrimSpace(c.Text())
		err := h.settingsSvc.SetReminderTime(context.Background(), uid, t)
		if err != nil {
			return c.Send("❌ "+err.Error(), tb.ModeHTML)
		}
		h.state.Reset(uid)

		markup := &tb.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("⚙️ настройки", "settings"), markup.Data("🏠 в меню", "menu")))
		return c.Send("✅ время напоминания установлено на <b>"+escapeHTML(t)+"</b>", markup, tb.ModeHTML)

	default:
		return c.Send("❌ непонятная команда. используй /start.", tb.ModeHTML)
	}
}

func (h *EntryHandler) handleNewEntry(c tb.Context) error {
	uid := c.Sender().ID
	h.state.Set(uid, &state.UserState{State: state.Creating})

	markup := &tb.ReplyMarkup{}
	btnCancel := markup.Data("❌ отмена", "cancel")
	markup.Inline(markup.Row(btnCancel))

	return c.Edit("✍️ <b>напиши свою запись:</b>", markup, tb.ModeHTML)
}

func (h *EntryHandler) handleCancel(c tb.Context) error {
	uid := c.Sender().ID
	h.state.Reset(uid)

	markup := &tb.ReplyMarkup{}
	btnMenu := markup.Data("🏠 в меню", "menu")
	markup.Inline(markup.Row(btnMenu))

	return c.Edit("❌ отменено.", markup, tb.ModeHTML)
}

func (h *EntryHandler) handleMenu(c tb.Context) error {
	h.state.Reset(c.Sender().ID)

	markup := &tb.ReplyMarkup{}
	markup.Inline(
		markup.Row(markup.Data("📝 новая запись", "new"), markup.Data("🎲 сюрприз", "random")),
		markup.Row(markup.Data("📋 мои записи", "list", "1"), markup.Data("⚙️ настройки", "settings")),
	)

	return c.Edit("📔 <b>дневник</b>\n\nтвой личный дневник в telegram. пиши записи, ищи их и получай напоминания.", markup, tb.ModeHTML)
}

func (h *EntryHandler) answer(c tb.Context, err error) error {
	if err != nil {
		c.Respond(&tb.CallbackResponse{Text: "❌ " + err.Error()})
	} else {
		c.Respond(&tb.CallbackResponse{})
	}
	return err
}

func (h *EntryHandler) handleNoop(c tb.Context) error {
	return nil
}

func firstWords(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max]) + "..."
}

func (h *EntryHandler) handleList(c tb.Context) error {
	uid := c.Sender().ID
	page := parseInt(c.Data(), 1)

	entries, total, err := h.svc.ListByUser(context.Background(), uid, page, pageSize)
	if err != nil {
		return c.Edit("❌ ошибка загрузки записей.", tb.ModeHTML)
	}

	markup := &tb.ReplyMarkup{}

	if total == 0 {
		markup.Inline(
			markup.Row(markup.Data("📝 новая запись", "new")),
			markup.Row(markup.Data("🏠 в меню", "menu")),
		)
		return c.Edit("📋 <b>Мои записи</b>\n\nу тебя пока нет записей. напиши первую! ✍️", markup, tb.ModeHTML)
	}

	totalPages := (total + pageSize - 1) / pageSize
	msg := fmt.Sprintf("📋 <b>Мои записи</b>  %d/%d", page, totalPages)

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
		label := firstWords(e.Content, 20) + " 📅 " + e.CreatedAt.Format("02.01")
		rows = append(rows, markup.Row(markup.Data(label, "view", strconv.Itoa(e.ID))))
	}

	rows = append(rows, markup.Row(
		markup.Data("📝 новая запись", "new"),
		markup.Data("🔍 поиск", "search_start"),
		markup.Data("🏠 в меню", "menu"),
	))

	markup.Inline(rows...)
	return c.Edit(msg, markup, tb.ModeHTML)
}

func (h *EntryHandler) handleView(c tb.Context) error {
	uid := c.Sender().ID
	id, _ := strconv.Atoi(c.Data())

	entry, err := h.svc.GetByID(context.Background(), uid, id)
	if err != nil {
		return c.Edit("❌ запись не найдена.", tb.ModeHTML)
	}

	var msg string
	if entry.CreatedAt.Equal(entry.UpdatedAt) {
		msg = fmt.Sprintf("<b>#%d</b>\n\n%s\n\n📅 <i>создана: %s</i>",
			entry.ID, escapeHTML(entry.Content),
			entry.CreatedAt.Format("02.01.2006 15:04"))
	} else {
		msg = fmt.Sprintf("<b>#%d</b>\n\n%s\n\n📅 <i>создана: %s</i>\n🔄 <i>изменена: %s</i>",
			entry.ID, escapeHTML(entry.Content),
			entry.CreatedAt.Format("02.01.2006 15:04"),
			entry.UpdatedAt.Format("02.01.2006 15:04"))
	}

	markup := &tb.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.Data("✏️ редактировать", "edit", strconv.Itoa(entry.ID)),
			markup.Data("🗑 удалить", "delete", strconv.Itoa(entry.ID)),
		),
		markup.Row(
			markup.Data("⬅️ назад", "list", "1"),
			markup.Data("🏠 в меню", "menu"),
		),
	)

	return c.Edit(msg, markup, tb.ModeHTML)
}

func (h *EntryHandler) handleEdit(c tb.Context) error {
	uid := c.Sender().ID
	id, _ := strconv.Atoi(c.Data())

	entry, err := h.svc.GetByID(context.Background(), uid, id)
	if err != nil {
		return c.Edit("❌ запись не найдена.", tb.ModeHTML)
	}

	h.state.Set(uid, &state.UserState{State: state.Editing, EditEntryID: id})

	markup := &tb.ReplyMarkup{}
	btnCancel := markup.Data("❌ отмена", "cancel")
	markup.Inline(markup.Row(btnCancel))

	msg := fmt.Sprintf("✏️ <b>редактирование #%d</b>\n\n<code>%s</code>\n\n<i>для редактирования записи нажми на текст выше вставь в поле ввода и отправляй</i>",
		entry.ID, escapeHTML(entry.Content))

	return c.Edit(msg, markup, tb.ModeHTML)
}

func (h *EntryHandler) handleDeleteConfirm(c tb.Context) error {
	id, _ := strconv.Atoi(c.Data())

	markup := &tb.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.Data("✅ да, удалить", "delete_yes", strconv.Itoa(id)),
			markup.Data("❌ нет", "list", "1"),
		),
	)

	return c.Edit("🗑 <b>точно удалить?</b>", markup, tb.ModeHTML)
}

func (h *EntryHandler) handleDeleteExec(c tb.Context) error {
	uid := c.Sender().ID
	id, _ := strconv.Atoi(c.Data())

	err := h.svc.Delete(context.Background(), uid, id)
	if err != nil {
		return c.Edit("❌ ошибка при удалении: "+err.Error(), tb.ModeHTML)
	}

	markup := &tb.ReplyMarkup{}
	markup.Inline(
		markup.Row(markup.Data("📋 к списку", "list", "1"), markup.Data("🏠 в меню", "menu")),
	)

	return c.Edit("🗑 запись удалена.", markup, tb.ModeHTML)
}

func (h *EntryHandler) handleRandom(c tb.Context) error {
	uid := c.Sender().ID

	entry, err := h.svc.RandomByUser(context.Background(), uid)
	if err != nil {
		return c.Edit("❌ у тебя пока нет записей.", tb.ModeHTML)
	}

	msg := fmt.Sprintf("🎲 <b>случайная запись #%d</b>\n\n%s\n\n📅 <i>%s</i>",
		entry.ID, escapeHTML(entry.Content), entry.CreatedAt.Format("02.01.2006 15:04"))

	markup := &tb.ReplyMarkup{}
	markup.Inline(
		markup.Row(
			markup.Data("👁 просмотр", "view", strconv.Itoa(entry.ID)),
			markup.Data("🎲 ещё", "random"),
			markup.Data("🏠 в меню", "menu"),
		),
	)

	return c.Edit(msg, markup, tb.ModeHTML)
}
