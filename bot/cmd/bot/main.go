package main

import (
	"context"
	"log"
	"os"
	"time"

	tb "gopkg.in/telebot.v3"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/dnevnik-bot/internal/handler"
	"github.com/user/dnevnik-bot/internal/plugin"
	"github.com/user/dnevnik-bot/internal/plugin/builtin"
	"github.com/user/dnevnik-bot/internal/repository"
	"github.com/user/dnevnik-bot/internal/service"
	"github.com/user/dnevnik-bot/internal/state"
)

// setBotAppearance registers the bot's commands and description so its profile
// looks polished in the Telegram UI. Failures are non-fatal (cosmetic only).
func setBotAppearance(bot *tb.Bot) {
	if err := bot.SetCommands([]tb.Command{
		{Text: "start", Description: "открыть дневник"},
	}); err != nil {
		log.Printf("setMyCommands failed: %v", err)
	}
	if err := bot.SetMyShortDescription("личный дневник в telegram", ""); err != nil {
		log.Printf("setMyShortDescription failed: %v", err)
	}
	desc := "твой личный дневник в telegram: пиши записи, веди разные дневники и получай напоминания."
	if err := bot.SetMyDescription(desc, ""); err != nil {
		log.Printf("setMyDescription failed: %v", err)
	}
}

func startReminderScheduler(bot *tb.Bot, settingsRepo *repository.PgSettings) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	notified := make(map[int64]struct{})
	var lastDate time.Time

	for now := range ticker.C {
		if now.Day() != lastDate.Day() || now.Month() != lastDate.Month() || now.Year() != lastDate.Year() {
			notified = make(map[int64]struct{})
		}
		lastDate = now

		users, err := settingsRepo.GetAllReminderUsers(context.Background())
		if err != nil {
			log.Printf("Reminder check error: %v", err)
			continue
		}

		currentTime := now.Format("15:04")
		for _, u := range users {
			if _, ok := notified[u.UserID]; ok {
				continue
			}
			if u.ReminderTime > currentTime {
				continue
			}
			notified[u.UserID] = struct{}{}

			hasEntry, err := settingsRepo.HasEntryToday(context.Background(), u.UserID, now)
			if err != nil {
				log.Printf("Error checking entry for %d: %v", u.UserID, err)
				notified[u.UserID] = struct{}{}
				continue
			}
			if hasEntry {
				continue
			}

			markup := &tb.ReplyMarkup{}
			markup.Inline(markup.Row(
				markup.Data("📝 написать", "new"),
				markup.Data("🏠 в меню", "menu"),
			))
			_, err = bot.Send(&tb.User{ID: u.UserID},
				"⏰ <b>напоминание!</b>\n\nты ещё не написал сегодня ни одной записи.",
				markup, tb.ModeHTML)
			if err != nil {
				log.Printf("Failed to send reminder to %d: %v", u.UserID, err)
			}
		}
	}
}

// buildRegistry wires every diary plugin into a single registry. To add a new
// diary type, create a plugin in internal/plugin/builtin, give it its own table
// and register it here.
func buildRegistry(pool *pgxpool.Pool) *plugin.Registry {
	registry := plugin.NewRegistry()
	registry.Register(builtin.NewFreeText(repository.NewDiaryStore(pool, "diary_freetext")))
	registry.Register(builtin.NewMood(repository.NewDiaryStore(pool, "diary_mood")))
	registry.Register(builtin.NewEmotions(repository.NewDiaryStore(pool, "diary_emotions")))
	return registry
}

// diaryTables returns the physical table name of every registered diary, used
// by cross-diary checks such as the daily reminder.
func diaryTables(registry *plugin.Registry) []string {
	all := registry.All()
	tables := make([]string, 0, len(all))
	for _, p := range all {
		tables = append(tables, p.Store().TableName())
	}
	return tables
}

func main() {
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("BOT_TOKEN is required")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	bot, err := tb.NewBot(tb.Settings{
		Token:  botToken,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatalf("Unable to create bot: %v", err)
	}

	setBotAppearance(bot)

	registry := buildRegistry(pool)

	stateManager := state.NewManager()
	entrySvc := service.NewEntryService(registry)
	settingsRepo := repository.NewPgSettings(pool, diaryTables(registry))
	settingsSvc := service.NewSettingsService(settingsRepo)
	pluginRepo := repository.NewPgPluginRepo(pool)
	diarySvc := service.NewDiaryService(registry, pluginRepo)
	entryHandler := handler.NewEntryHandler(entrySvc, settingsSvc, diarySvc, stateManager, bot)
	entryHandler.Register()

	go startReminderScheduler(bot, settingsRepo)

	log.Println("Bot started!")
	bot.Start()
}
