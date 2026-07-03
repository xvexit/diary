package main

import (
	"context"
	"log"
	"os"
	"time"

	tb "gopkg.in/telebot.v3"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/user/dnevnik-bot/internal/handler"
	"github.com/user/dnevnik-bot/internal/repository"
	"github.com/user/dnevnik-bot/internal/service"
	"github.com/user/dnevnik-bot/internal/state"
)

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

	stateManager := state.NewManager()
	entryRepo := repository.NewPgEntry(pool)
	entrySvc := service.NewEntryService(entryRepo)
	entryHandler := handler.NewEntryHandler(entrySvc, stateManager, bot)
	entryHandler.Register()

	log.Println("Bot started!")
	bot.Start()
}
