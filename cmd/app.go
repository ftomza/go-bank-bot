package main

import (
	"context"
	"log"
	"os"

	"golang.org/x/oauth2/google"

	"github.com/ftomza/go-bank-bot/pkg/bot"
	"github.com/ftomza/go-bank-bot/pkg/store"
	"gopkg.in/tucnak/telebot.v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {

	db, err := gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	userRepo := store.NewGormUserRepository(db)
	err = userRepo.Migration(context.Background())
	if err != nil {
		log.Fatalf("migration db: %v", err)
	}

	cred := os.Getenv("CREDENTIALS")
	if cred == "" {
		log.Fatalf("CREDENTIALS not set")
	}

	config, err := google.ConfigFromJSON([]byte(os.Getenv("CREDENTIALS")),
		"https://www.googleapis.com/auth/drive",
		"https://www.googleapis.com/auth/drive.file",
		"https://www.googleapis.com/auth/spreadsheets",
	)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	token := os.Getenv("TOKEN")
	if cred == "" {
		log.Fatalf("TOKEN not set")
	}

	debug := os.Getenv("DEBUG")

	tb, _ := telebot.NewBot(telebot.Settings{
		Token:   token,
		Poller:  bot.NewPoller(10),
		Verbose: debug != "",
	})

	b := bot.NewTelegramBot(tb, userRepo, store.NewGoogleClient(config))

	b.Start()
}
