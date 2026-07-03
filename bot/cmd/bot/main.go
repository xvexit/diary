package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	tb "gopkg.in/telebot.v3"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/dnevnik-bot/internal/handler"
	"github.com/user/dnevnik-bot/internal/repository"
	"github.com/user/dnevnik-bot/internal/service"
	"github.com/user/dnevnik-bot/internal/state"
)

func startReminderScheduler(bot *tb.Bot, settingsRepo *repository.PgSettings) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		userIDs, err := settingsRepo.GetUsersDueForReminder(context.Background(), now)
		if err != nil {
			log.Printf("Reminder check error: %v", err)
			continue
		}

		for _, uid := range userIDs {
			markup := &tb.ReplyMarkup{}
			markup.Inline(markup.Row(markup.Data("📝 Новая запись", "new")))
			_, err := bot.Send(&tb.User{ID: uid},
				"⏰ <b>Напоминание!</b>\n\nТы ещё не написал сегодня в дневник! 📝",
				markup, tb.ModeHTML)
			if err != nil {
				log.Printf("Failed to send reminder to %d: %v", uid, err)
			}
		}
	}
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

	httpClient := &http.Client{}
	if proxyURL := os.Getenv("BOT_PROXY"); proxyURL != "" {
		u, err := url.Parse(proxyURL)
		if err != nil {
			log.Fatalf("Invalid BOT_PROXY: %v", err)
		}
		httpClient.Transport = &http.Transport{Proxy: http.ProxyURL(u)}
		log.Printf("Using proxy: %s", proxyURL)
	}

	bot, err := tb.NewBot(tb.Settings{
		Token:  botToken,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
		Client: httpClient,
	})
	if err != nil {
		log.Fatalf("Unable to create bot: %v", err)
	}

	stateManager := state.NewManager()
	entryRepo := repository.NewPgEntry(pool)
	entrySvc := service.NewEntryService(entryRepo)
	settingsRepo := repository.NewPgSettings(pool)
	settingsSvc := service.NewSettingsService(settingsRepo)
	entryHandler := handler.NewEntryHandler(entrySvc, settingsSvc, stateManager, bot)
	entryHandler.Register()

	go startReminderScheduler(bot, settingsRepo)

	log.Println("Bot started!")
	bot.Start()
}
