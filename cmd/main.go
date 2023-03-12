package main

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/kostage/cosmos_voter/internal/app"
	"github.com/kostage/cosmos_voter/internal/cmdrunner"
	"github.com/kostage/cosmos_voter/internal/config"
	"github.com/kostage/cosmos_voter/internal/tgbot"
	"github.com/kostage/cosmos_voter/internal/vote"
)

const (
	configFile = "config.yaml"
)

func main() {
	conf, err := config.ParseConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}
	bot, err := tgbot.NewTgBot(conf.BotToken)
	if err != nil {
		log.Fatal(err)
	}
	runner := cmdrunner.NewCmdRunner()
	voter := vote.NewCosmosVoter(
		runner,
		conf.DaemonPath,
		conf.KeyChainPass,
		conf.VoterWallet,
		conf.Fees,
		conf.ChainId,
	)
	app := app.NewApp(voter, bot, conf.AllowedUser)
	if err := app.Run(context.TODO()); err != nil {
		log.Fatal(err)
	}
}
