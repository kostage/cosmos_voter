package app

// import (
// 	"context"
// 	"fmt"
// 	"testing"
// 	"time"

// 	"github.com/golang/mock/gomock"
// 	"github.com/kostage/cosmos_voter/internal/config"
// 	"github.com/kostage/cosmos_voter/internal/tgbot"
// 	"github.com/kostage/cosmos_voter/internal/vote"
// 	"github.com/stretchr/testify/assert"
// )

// func TestApp_GetPropsFailed(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	bot, err := tgbot.NewTgBot(&config.TgConfig{""})
// 	assert.NoError(t, err)

// 	voter := vote.NewMockVoter(ctrl)
// 	voterErr := fmt.Errorf("get proposals failed")
// 	voter.EXPECT().GetVoting().Return(nil, voterErr)

// 	app := NewApp(voter, bot)
// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
// 	defer cancel()
// 	assert.ErrorIs(t, app.Run(ctx), voterErr)
// }

// func TestApp_VoteFailed(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	bot, err := tgbot.NewTgBot(&config.TgConfig{""})
// 	assert.NoError(t, err)

// 	dummyProps := []vote.Proposal{
// 		{
// 			Id:          1,
// 			Title:       "Dummy proposal",
// 			Description: "Dummy proposal",
// 			VotedYes:    70,
// 			VotedNo:     30,
// 		},
// 		{
// 			Id:          2,
// 			Title:       "Dummy proposal\nmultiline",
// 			Description: "Dummy proposal\nmultiline",
// 			VotedYes:    70,
// 			VotedNo:     30,
// 		},
// 	}
// 	voter := vote.NewMockVoter(ctrl)
// 	voter.EXPECT().GetVoting().Return(dummyProps, nil)

// 	voterErr := fmt.Errorf("vote failed")
// 	voter.EXPECT().Vote(1, true).Return(voterErr)

// 	app := NewApp(voter, bot)
// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
// 	defer cancel()
// 	assert.NoError(t, app.Run(ctx))
// }

// func TestApp_Manual(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	bot, err := tgbot.NewTgBot(&config.TgConfig{""})
// 	assert.NoError(t, err)

// 	dummyProps := []vote.Proposal{
// 		{
// 			Id:          1,
// 			Title:       "Dummy proposal",
// 			Description: "Dummy proposal",
// 			VotedYes:    70,
// 			VotedNo:     30,
// 		},
// 		{
// 			Id:          2,
// 			Title:       "Dummy proposal\nmultiline",
// 			Description: "Dummy proposal\nmultiline",
// 			VotedYes:    70,
// 			VotedNo:     30,
// 		},
// 	}
// 	voter := vote.NewMockVoter(ctrl)
// 	voter.EXPECT().GetVoting().Return(dummyProps, nil).AnyTimes()
// 	voter.EXPECT().Vote(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

// 	app := NewApp(voter, bot)
// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
// 	defer cancel()
// 	assert.NoError(t, app.Run(ctx))
// }
