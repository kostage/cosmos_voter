package tgbot

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type TgBot struct {
	*tgbotapi.BotAPI
	username string
}

func NewTgBot(token, username string) (*TgBot, error) {
	// Create a new bot with your token
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create tg bot API")
	}
	return &TgBot{
		BotAPI:   api,
		username: ("@" + username),
	}, nil
}

func (b *TgBot) ProcessUpdates(
	ctx context.Context,
	handler func(tgbotapi.Update) error,
) error {
	// Set up a handler for the /start command
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := b.BotAPI.GetUpdatesChan(u)
	if err != nil {
		return errors.Wrap(err, "failed to open tg updates channel")
	}

	for {
		select {
		case update, ok := <-updates:
			if !ok {
				return fmt.Errorf("updates chan closed")
			}
			if ok := b.validateUser(update); !ok {
				continue
			}
			if err := handler(update); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (b *TgBot) validateUser(update tgbotapi.Update) bool {
	if update.Message.From == nil {
		log.Error("skipped update from unknown user")
		return false
	}
	if update.Message.From.UserName != b.username {
		log.Errorf("skipped update from some motherfucker: %s", update.Message.From.UserName)
		return false
	}
	return true
}
