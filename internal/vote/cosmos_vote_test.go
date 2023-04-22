package vote

import (
	"context"
	"fmt"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/kostage/cosmos_voter/internal/cmdrunner"
	"github.com/stretchr/testify/assert"

	_ "embed"
)

//go:embed example_proposals.json
var example_proposals []byte

//go:embed example_vote_291.json
var example_vote_291 []byte

//go:embed example_vote_294.json
var example_vote_294 []byte

//go:embed example_vote_295.json
var example_vote_295 []byte

//go:embed example_tally.json
var example_tally []byte

//go:embed example_validators.yaml
var example_validators []byte

func TestGetCosmosProposals(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := cmdrunner.NewMockCmdRunner(ctrl)
	defRunnerFactory = func() cmdrunner.CmdRunner { return runner }
	expectedPropArgs := []string{"query", "gov", "proposals", "--status", "VotingPeriod", "-o", "json"}
	expectedTallyArgs1 := []string{"query", "gov", "tally", "291", "-o", "json"}
	expectedTallyArgs2 := []string{"query", "gov", "tally", "294", "-o", "json"}
	expectedTallyArgs3 := []string{"query", "gov", "tally", "295", "-o", "json"}
	expectedVoteArgs1 := []string{"query", "gov", "vote", "291", "-o", "json"}
	expectedVoteArgs2 := []string{"query", "gov", "vote", "294", "-o", "json"}
	expectedVoteArgs3 := []string{"query", "gov", "vote", "295", "-o", "json"}
	expectedGetValidatotsArgs := []string{"query", "tendermint-validator-set"}
	runner.EXPECT().Run(gomock.Any(), "daemon", expectedPropArgs, nil).Return(example_proposals, nil, nil)
	runner.EXPECT().Run(gomock.Any(), "daemon", expectedTallyArgs1, nil).Return(example_tally, nil, nil)
	runner.EXPECT().Run(gomock.Any(), "daemon", expectedTallyArgs2, nil).Return(example_tally, nil, nil)
	runner.EXPECT().Run(gomock.Any(), "daemon", expectedTallyArgs3, nil).Return(example_tally, nil, nil)
	runner.EXPECT().Run(gomock.Any(), "daemon", expectedVoteArgs1, nil).Return(example_vote_291, nil, nil)
	runner.EXPECT().Run(gomock.Any(), "daemon", expectedVoteArgs2, nil).Return(example_vote_294, nil, nil)
	runner.EXPECT().Run(gomock.Any(), "daemon", expectedVoteArgs3, nil).Return(example_vote_295, nil, nil)
	runner.EXPECT().Run(gomock.Any(), "daemon", expectedGetValidatotsArgs, nil).Return(example_validators, nil, nil)

	voter := NewCosmosVoter("daemon", "password", "", "", "")
	proposals, err := voter.GetVoting(context.Background())
	assert.NoError(t, err)
	assert.Len(t, proposals, 3)
}

func TestGetCosmosVoted(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := cmdrunner.NewMockCmdRunner(ctrl)
	defRunnerFactory = func() cmdrunner.CmdRunner { return runner }
	expectedArgs := []string{"query", "gov", "vote", "1", "voterWallet", "-o", "json"}
	runner.EXPECT().Run(gomock.Any(), "daemon", expectedArgs, nil).Return(example_vote_294, nil, nil)

	voter := NewCosmosVoter("daemon", "password", "voterWallet", "", "")
	voted, err := voter.HasVoted(context.Background(), "1")
	assert.NoError(t, err)
	assert.False(t, voted)
}

func TestGetCosmosNotVoted(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := cmdrunner.NewMockCmdRunner(ctrl)
	defRunnerFactory = func() cmdrunner.CmdRunner { return runner }
	expectedArgs := []string{"query", "gov", "vote", "1", "voterWallet", "-o", "json"}
	runner.EXPECT().Run(gomock.Any(), "daemon", expectedArgs, nil).Return(nil, nil, fmt.Errorf("somerr"))

	voter := NewCosmosVoter("daemon", "password", "voterWallet", "", "")
	voted, err := voter.HasVoted(context.Background(), "1")
	assert.NoError(t, err)
	assert.False(t, voted)
}

func TestGetCosmosVotedParseFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	runner := cmdrunner.NewMockCmdRunner(ctrl)
	defRunnerFactory = func() cmdrunner.CmdRunner { return runner }
	expectedArgs := []string{"query", "gov", "vote", "1", "voterWallet", "-o", "json"}
	runner.EXPECT().Run(gomock.Any(), "daemon", expectedArgs, nil).Return([]byte("not json"), nil, nil)

	voter := NewCosmosVoter("daemon", "password", "voterWallet", "", "")
	voted, err := voter.HasVoted(context.Background(), "1")
	assert.Error(t, err)
	assert.False(t, voted)
}
