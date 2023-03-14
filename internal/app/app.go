package app

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/kostage/cosmos_voter/internal/tgbot"
	"github.com/kostage/cosmos_voter/internal/vote"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	cmdTimeout = time.Second * 15

	votePromptTmplFile = "votePrompt.tmpl"
	voteButtonData     = "vote %s on %s"
)

var (
	votePromptTmpl = template.Must(template.ParseFiles(votePromptTmplFile))
)

type App struct {
	voter    vote.Voter
	bot      *tgbot.TgBot
	username string
}

func NewApp(voter vote.Voter, bot *tgbot.TgBot, username string) *App {
	return &App{
		voter:    voter,
		bot:      bot,
		username: username,
	}
}

func (app *App) Run(ctx context.Context) error {
	return app.bot.ProcessUpdates(
		ctx,
		func(update tgbotapi.Update) error {
			if update.Message != nil && update.Message.IsCommand() {
				if err := app.ProcessCommand(ctx, update); err != nil {
					return errors.Wrapf(err, "failed to process command '%s'", update.Message.Command())
				}
				return nil
			}
			if update.CallbackQuery == nil {
				return nil
			}

			if err := app.ProcessVoteCallback(ctx, update); err != nil {
				return errors.Wrapf(err, "failed to process vote callback '%s'", update.CallbackQuery.Data)
			}
			return nil
		},
	)
}

func (app *App) ProcessCommand(ctx context.Context, update tgbotapi.Update) error {
	if update.Message.Command() != "start" {
		errResp := fmt.Sprintf("Unknown command: %s", update.Message.Command())
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, errResp)
		if _, err := app.bot.BotAPI.Send(msg); err != nil {
			return errors.Wrap(err, "failed to send tg message")
		}
	}
	log.Info("received start")
	if ok := app.validateUser(update); !ok {
		log.Error("skipping command")
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, cmdTimeout)
	defer cancel()
	proposals, err := app.voter.GetVoting(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get proposals")
	}
	if len(proposals) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "No active proposals found")
		if _, err := app.bot.BotAPI.Send(msg); err != nil {
			log.Info("found 0 proposals")
			return errors.Wrap(err, "failed to send msg")
		}
	}
	for _, prop := range proposals {
		log.Infof("found proposal: %s", prop.Id)
		if voted, _ := app.voter.HasVoted(ctx, prop.Id); voted {
			log.Infof("skipped already voted proposal %s", prop.Id)
			continue
		}
		if err := app.SendVotePrompt(prop, update.Message.Chat.ID); err != nil {
			log.Infof("sent prompt for proposal: %s", prop.Id)
			return errors.Wrap(err, "failed to send vote prompt")
		}
	}
	return nil
}

func (app *App) SendVotePrompt(prop vote.Proposal, chatID int64) error {
	// Create the keyboard with two buttons
	yesButton := tgbotapi.NewInlineKeyboardButtonData("Yes", fmt.Sprintf(voteButtonData, "yes", prop.Id))
	noButton := tgbotapi.NewInlineKeyboardButtonData("No", fmt.Sprintf(voteButtonData, "no", prop.Id))
	skipButton := tgbotapi.NewInlineKeyboardButtonData("Skip", fmt.Sprintf(voteButtonData, "skip", prop.Id))
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		[]tgbotapi.InlineKeyboardButton{yesButton, noButton, skipButton},
	)

	// Send the message to the user
	promptBuf := &bytes.Buffer{}
	if err := votePromptTmpl.Execute(promptBuf, prop); err != nil {
		return errors.Wrap(err, "failed to send vote keyboard")
	}
	msg := tgbotapi.NewMessage(chatID, promptBuf.String())
	if _, err := app.bot.BotAPI.Send(msg); err != nil {
		return errors.Wrap(err, "failed to send vote prompt")
	}

	// Send the keyboard to the user
	msg = tgbotapi.NewMessage(chatID, "Please vote yes, no or skip for now")
	msg.ReplyMarkup = keyboard
	if _, err := app.bot.BotAPI.Send(msg); err != nil {
		return errors.Wrap(err, "failed to send vote keyboard")
	}
	return nil
}

func (app *App) ProcessVoteCallback(ctx context.Context, update tgbotapi.Update) error {
	reportErr := func(err error) error {
		errText := fmt.Sprintf("Failed to process callback data '%s', err: %v", update.CallbackQuery.Data, err)
		msg := tgbotapi.NewEditMessageText(
			update.CallbackQuery.Message.Chat.ID,
			update.CallbackQuery.Message.MessageID,
			errText,
		)
		if _, err := app.bot.BotAPI.Send(msg); err != nil {
			return errors.Wrapf(err, "failed to send tg message '%s'", errText)
		}
		callbackAnswer := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
		if _, err := app.bot.BotAPI.AnswerCallbackQuery(callbackAnswer); err != nil {
			return errors.Wrap(err, "failed to answer the callback query to remove the 'loading' animation from the button")
		}
		return nil
	}
	log.Infof("received callback: %s", update.CallbackQuery.Data)
	var voteStr string
	var propID string
	if _, err := fmt.Sscanf(update.CallbackQuery.Data, voteButtonData, &voteStr, &propID); err != nil {
		return reportErr(err)
	}
	switch voteStr {
	case "yes":
	case "no":
	case "skip":
	default:
		log.Errorf("vote is not [yes|no|skip] in callback '%s'", update.CallbackQuery.Data)
		return reportErr(fmt.Errorf("vote is not [yes|no|skip]"))
	}
	if voteStr != "skip" {
		ctx, cancel := context.WithTimeout(ctx, cmdTimeout)
		defer cancel()
		if err := app.voter.Vote(ctx, propID, voteStr); err != nil {
			return reportErr(errors.Wrap(err, "vote failed"))
		}
	}
	congrat := fmt.Sprintf("You voted %s on proposal %s", voteStr, propID)
	msg := tgbotapi.NewEditMessageText(
		update.CallbackQuery.Message.Chat.ID,
		update.CallbackQuery.Message.MessageID,
		congrat,
	)
	if _, err := app.bot.BotAPI.Send(msg); err != nil {
		return errors.Wrapf(err, "failed to send tg message '%s'", congrat)
	}
	callbackAnswer := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
	if _, err := app.bot.BotAPI.AnswerCallbackQuery(callbackAnswer); err != nil {
		return errors.Wrap(err, "failed to answer the callback query to remove the 'loading' animation from the button")
	}
	log.Infof("voted %s on proposal %s", voteStr, propID)
	return nil
}

func (app *App) validateUser(update tgbotapi.Update) bool {
	if update.Message.From == nil {
		log.Error("unknown user")
		return false
	}
	if update.Message.From.UserName != app.username {
		log.Errorf("command from some motherfucker: %s", update.Message.From.UserName)
		return false
	}
	return true
}
