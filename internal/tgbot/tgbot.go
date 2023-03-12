package tgbot

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
)

type TgBot struct {
	*tgbotapi.BotAPI
}

func NewTgBot(token string) (*TgBot, error) {
	// Create a new bot with your token
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create tg bot API")
	}
	return &TgBot{
		BotAPI: api,
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
			if err := handler(update); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}
