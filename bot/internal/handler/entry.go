package handler

import (
	"context"
	"fmt"
	"strings"

	tb "gopkg.in/telebot.v3"

	"github.com/user/dnevnik-bot/internal/service"
	"github.com/user/dnevnik-bot/internal/state"
)

const pageSize = 5

type EntryHandler struct {
	svc   *service.EntryService
	state *state.Manager
	bot   *tb.Bot
}

func NewEntryHandler(svc *service.EntryService, st *state.Manager, bot *tb.Bot) *EntryHandler {
	return &EntryHandler{
		svc:   svc,
		state: st,
		bot:   bot,
	}
}

func (h *EntryHandler) Register() {
	h.bot.Handle("/start", h.handleStart)
	h.bot.Handle(tb.OnText, h.handleText)
	h.bot.Handle(tb.OnCallback, func(c tb.Context) error {
		data := c.Data()
		switch {
		case data == "new":
			return h.handleNewEntry(c)
		case data == "cancel":
			return h.handleCancel(c)
		case data == "menu":
			return h.handleMenu(c)
		case data == "noop":
			return h.handleNoop(c)
		case data == "random":
			return h.handleRandom(c)
		case strings.HasPrefix(data, "list:"):
			return h.handleList(c)
		case strings.HasPrefix(data, "view:"):
			return h.handleView(c)
		case strings.HasPrefix(data, "edit:"):
			return h.handleEdit(c)
		case strings.HasPrefix(data, "delete_yes:"):
			return h.handleDeleteExec(c)
		case strings.HasPrefix(data, "delete:"):
			return h.handleDeleteConfirm(c)
		}
		return nil
	})
}

func parsePage(data string) int {
	page := 1
	fmt.Sscanf(data, "list:%d", &page)
	if page < 1 {
		page = 1
	}
	return page
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func (h *EntryHandler) handleStart(c tb.Context) error {
	h.state.Reset(c.Sender().ID)

	msg := "📔 <b>Дневник</b>\n\nТвой личный дневник в Telegram. Пиши записи, смотри их с удобной навигацией."

	markup := &tb.ReplyMarkup{}
	btnNew := markup.Data("📝 Новая запись", "new")
	btnList := markup.Data("📋 Мои записи", "list:1")
	btnRandom := markup.Data("🎲 Сюрприз", "random")
	markup.Inline(markup.Row(btnNew), markup.Row(btnList), markup.Row(btnRandom))

	return c.Send(msg, markup, tb.ModeHTML)
}

func (h *EntryHandler) handleText(c tb.Context) error {
	uid := c.Sender().ID
	us := h.state.Get(uid)

	switch us.State {
	case state.Creating:
		content := strings.TrimSpace(c.Text())
		if content == "" {
			return c.Send("❌ Запись не может быть пустой.", tb.ModeHTML)
		}
		entry, err := h.svc.Create(context.Background(), uid, content)
		if err != nil {
			return c.Send("❌ Ошибка при сохранении: "+err.Error(), tb.ModeHTML)
		}
		h.state.Reset(uid)

		markup := &tb.ReplyMarkup{}
		btnList := markup.Data("📋 К списку", "list:1")
		btnMenu := markup.Data("🏠 В меню", "menu")
		markup.Inline(markup.Row(btnList, btnMenu))

		return c.Send(fmt.Sprintf("✅ Запись #%d сохранена!", entry.ID), markup, tb.ModeHTML)

	case state.Editing:
		content := strings.TrimSpace(c.Text())
		if content == "" {
			return c.Send("❌ Запись не может быть пустой.", tb.ModeHTML)
		}
		err := h.svc.Update(context.Background(), uid, us.EditEntryID, content)
		if err != nil {
			return c.Send("❌ Ошибка при обновлении: "+err.Error(), tb.ModeHTML)
		}
		h.state.Reset(uid)

		markup := &tb.ReplyMarkup{}
		btnList := markup.Data("📋 К списку", "list:1")
		btnMenu := markup.Data("🏠 В меню", "menu")
		markup.Inline(markup.Row(btnList, btnMenu))

		return c.Send("✅ Запись обновлена!", markup, tb.ModeHTML)

	default:
		return c.Send("❌ Непонятная команда. Используй /start.", tb.ModeHTML)
	}
}

func (h *EntryHandler) handleNewEntry(c tb.Context) error {
	uid := c.Sender().ID
	h.state.Set(uid, &state.UserState{State: state.Creating})

	markup := &tb.ReplyMarkup{}
	btnCancel := markup.Data("❌ Отмена", "cancel")
	markup.Inline(markup.Row(btnCancel))

	return c.Edit("✍️ <b>Напиши свою запись:</b>", markup, tb.ModeHTML)
}

func (h *EntryHandler) handleCancel(c tb.Context) error {
	uid := c.Sender().ID
	h.state.Reset(uid)

	markup := &tb.ReplyMarkup{}
	btnMenu := markup.Data("🏠 В меню", "menu")
	markup.Inline(markup.Row(btnMenu))

	return c.Edit("❌ Отменено.", markup, tb.ModeHTML)
}

func (h *EntryHandler) handleMenu(c tb.Context) error {
	h.state.Reset(c.Sender().ID)

	markup := &tb.ReplyMarkup{}
	btnNew := markup.Data("📝 Новая запись", "new")
	btnList := markup.Data("📋 Мои записи", "list:1")
	btnRandom := markup.Data("🎲 Сюрприз", "random")
	markup.Inline(markup.Row(btnNew), markup.Row(btnList), markup.Row(btnRandom))

	return c.Edit("📔 <b>Дневник</b>\n\nТвой личный дневник в Telegram. Пиши записи, смотри их с удобной навигацией.", markup, tb.ModeHTML)
}

func (h *EntryHandler) handleNoop(c tb.Context) error {
	return c.Respond()
}

func (h *EntryHandler) handleList(c tb.Context) error {
	uid := c.Sender().ID
	page := parsePage(c.Data())

	entries, total, err := h.svc.ListByUser(context.Background(), uid, page, pageSize)
	if err != nil {
		return c.Edit("❌ Ошибка загрузки записей.", tb.ModeHTML)
	}

	markup := &tb.ReplyMarkup{}

	if total == 0 {
		btnNew := markup.Data("📝 Новая запись", "new")
		btnMenu := markup.Data("🏠 В меню", "menu")
		markup.Inline(markup.Row(btnNew), markup.Row(btnMenu))
		return c.Edit("📋 <b>Мои записи</b>\n\nУ тебя пока нет записей. Напиши первую! ✍️", markup, tb.ModeHTML)
	}

	totalPages := (total + pageSize - 1) / pageSize

	var b strings.Builder
	b.WriteString("📋 <b>Мои записи</b>\n\n")
	for _, e := range entries {
		runes := []rune(e.Content)
		preview := string(runes)
		if len(runes) > 50 {
			preview = string(runes[:50]) + "..."
		}
		b.WriteString(fmt.Sprintf("#%d 📅 %s — <i>%s</i>\n",
			e.ID, e.CreatedAt.Format("02.01.2006"), escapeHTML(preview)))
	}

	msg := b.String()

	var rows []tb.Row

	var navRow []tb.Btn
	if page > 1 {
		navRow = append(navRow, markup.Data("◀️", fmt.Sprintf("list:%d", page-1)))
	}
	navRow = append(navRow, markup.Data(fmt.Sprintf("%d/%d", page, totalPages), "noop"))
	if page < totalPages {
		navRow = append(navRow, markup.Data("▶️", fmt.Sprintf("list:%d", page+1)))
	}
	rows = append(rows, navRow)

	for _, e := range entries {
		rows = append(rows, markup.Row(markup.Data(fmt.Sprintf("#%d 👁 Просмотр", e.ID), fmt.Sprintf("view:%d", e.ID))))
	}

	rows = append(rows, markup.Row(markup.Data("📝 Новая запись", "new"), markup.Data("🏠 В меню", "menu")))

	markup.Inline(rows...)

	return c.Edit(msg, markup, tb.ModeHTML)
}

func (h *EntryHandler) handleView(c tb.Context) error {
	uid := c.Sender().ID
	var id int
	fmt.Sscanf(c.Data(), "view:%d", &id)

	entry, err := h.svc.GetByID(context.Background(), uid, id)
	if err != nil {
		return c.Edit("❌ Запись не найдена.", tb.ModeHTML)
	}

	msg := fmt.Sprintf("<b>#%d</b>\n\n%s\n\n📅 <i>Создана: %s</i>\n🔄 <i>Изменена: %s</i>",
		entry.ID, escapeHTML(entry.Content),
		entry.CreatedAt.Format("02.01.2006 15:04"),
		entry.UpdatedAt.Format("02.01.2006 15:04"))

	markup := &tb.ReplyMarkup{}
	btnEdit := markup.Data("✏️ Редактировать", fmt.Sprintf("edit:%d", entry.ID))
	btnDelete := markup.Data("🗑 Удалить", fmt.Sprintf("delete:%d", entry.ID))
	btnBack := markup.Data("⬅️ Назад", "list:1")
	btnMenu := markup.Data("🏠 В меню", "menu")
	markup.Inline(markup.Row(btnEdit, btnDelete), markup.Row(btnBack, btnMenu))

	return c.Edit(msg, markup, tb.ModeHTML)
}

func (h *EntryHandler) handleEdit(c tb.Context) error {
	uid := c.Sender().ID
	var id int
	fmt.Sscanf(c.Data(), "edit:%d", &id)

	entry, err := h.svc.GetByID(context.Background(), uid, id)
	if err != nil {
		return c.Edit("❌ Запись не найдена.", tb.ModeHTML)
	}

	h.state.Set(uid, &state.UserState{State: state.Editing, EditEntryID: id})

	markup := &tb.ReplyMarkup{}
	btnCancel := markup.Data("❌ Отмена", "cancel")
	markup.Inline(markup.Row(btnCancel))

	msg := fmt.Sprintf("✏️ <b>Редактирование #%d</b>\n\n<i>Текущий текст:</i>\n%s\n\nОтправь новый текст:",
		entry.ID, escapeHTML(entry.Content))

	return c.Edit(msg, markup, tb.ModeHTML)
}

func (h *EntryHandler) handleDeleteConfirm(c tb.Context) error {
	var id int
	fmt.Sscanf(c.Data(), "delete:%d", &id)

	markup := &tb.ReplyMarkup{}
	btnYes := markup.Data("✅ Да, удалить", fmt.Sprintf("delete_yes:%d", id))
	btnNo := markup.Data("❌ Нет", "list:1")
	markup.Inline(markup.Row(btnYes, btnNo))

	return c.Edit("🗑 <b>Точно удалить?</b>", markup, tb.ModeHTML)
}

func (h *EntryHandler) handleDeleteExec(c tb.Context) error {
	uid := c.Sender().ID
	var id int
	fmt.Sscanf(c.Data(), "delete_yes:%d", &id)

	err := h.svc.Delete(context.Background(), uid, id)
	if err != nil {
		return c.Edit("❌ Ошибка при удалении: "+err.Error(), tb.ModeHTML)
	}

	markup := &tb.ReplyMarkup{}
	btnList := markup.Data("📋 К списку", "list:1")
	btnMenu := markup.Data("🏠 В меню", "menu")
	markup.Inline(markup.Row(btnList, btnMenu))

	return c.Edit("🗑 Запись удалена.", markup, tb.ModeHTML)
}

func (h *EntryHandler) handleRandom(c tb.Context) error {
	uid := c.Sender().ID

	entry, err := h.svc.RandomByUser(context.Background(), uid)
	if err != nil {
		return c.Edit("❌ У тебя пока нет записей.", tb.ModeHTML)
	}

	msg := fmt.Sprintf("🎲 <b>Случайная запись #%d</b>\n\n%s\n\n📅 <i>%s</i>",
		entry.ID, escapeHTML(entry.Content), entry.CreatedAt.Format("02.01.2006 15:04"))

	markup := &tb.ReplyMarkup{}
	btnView := markup.Data("👁 Просмотр", fmt.Sprintf("view:%d", entry.ID))
	btnRandom := markup.Data("🎲 Ещё", "random")
	btnMenu := markup.Data("🏠 В меню", "menu")
	markup.Inline(markup.Row(btnView, btnRandom, btnMenu))

	return c.Edit(msg, markup, tb.ModeHTML)
}
