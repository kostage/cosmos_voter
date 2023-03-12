package vote

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kostage/cosmos_voter/internal/cmdrunner"
	log "github.com/sirupsen/logrus"
)

var (
	cosmosGetVotingCmdArgs = "query gov proposals --status VotingPeriod -o json"
	cosmosHasVotedCmdArgs  = "query gov vote %s %s -o json"
	cosmosTallyCmdArgs     = "query gov tally %s -o json"
	cosmosVoteCmdArgs      = "tx gov vote %s %s --from %s --fees %s --chain-id %s -y"
	cosmosNumVotesCmdArgs  = "query gov votes %s -o json"
)

type cosmosProposalsResponse struct {
	Proposals []cosmosProposal `json:"proposals"`
}

type cosmosProposal struct {
	ProposalID    string                `json:"proposal_id"`
	Content       cosmosProposalContent `json:"content"`
	VotingEndTime time.Time             `json:"voting_end_time"`
}

type cosmosProposalContent struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type cosmosHasVotedResponse struct {
	Option string `json:"option"`
}

type cosmosTallyResponse struct {
	Yes        int64 `json:"yes,string"`
	Abstain    int64 `json:"abstain,string"`
	No         int64 `json:"no,string"`
	NoWithVeto int64 `json:"no_with_veto,string"`
}

type cosmosNumVotesResponse struct {
	Votes []interface{} `json:"votes"`
}

type CosmosVoter struct {
	runner       cmdrunner.CmdRunner
	daemonPath   string
	keychainPass string
	voterWallet  string
	fees         string
	chainId      string
}

func NewCosmosVoter(
	runner cmdrunner.CmdRunner,
	daemonPath string,
	keychainPass string,
	voterWallet string,
	fees string,
	chainId string,
) *CosmosVoter {
	return &CosmosVoter{
		runner:       runner,
		daemonPath:   daemonPath,
		keychainPass: keychainPass,
		voterWallet:  voterWallet,
		fees:         fees,
		chainId:      chainId,
	}
}

func (cv *CosmosVoter) GetVoting(ctx context.Context) ([]Proposal, error) {
	args := strings.Fields(cosmosGetVotingCmdArgs)
	stdout, stderr, err := cv.runner.Run(
		ctx,
		cv.daemonPath,
		args,
		nil,
	)
	if err != nil {
		logCmdErr(cv.daemonPath, args, stdout, stderr, err)
		return nil, fmt.Errorf("failed to run cosmos proposals query: %v", err)
	}
	cosmosProposals := cosmosProposalsResponse{}
	if err := json.Unmarshal(stdout, &cosmosProposals); err != nil {
		logCmdErr(cv.daemonPath, args, stdout, stderr, err)
		return nil, fmt.Errorf("failed to unmarshal cosmos proposals: %v", err)
	}
	proposals := make([]Proposal, 0, len(cosmosProposals.Proposals))
	for _, cosmosProp := range cosmosProposals.Proposals {
		tally, err := cv.tally(ctx, cosmosProp.ProposalID)
		if err != nil {
			return nil, err
		}
		voted, _ := cv.numVoted(ctx, cosmosProp.ProposalID)
		all := tally.Yes + tally.No + tally.NoWithVeto
		yes := int(tally.Yes * 100 / all)
		no := int(tally.No * 100 / all)
		veto := int(tally.NoWithVeto * 100 / all)
		endsInHrs := cosmosProp.VotingEndTime.Sub(time.Now().UTC()).Hours()
		endsInHrs = math.Round(endsInHrs*100) / 100
		proposals = append(proposals, Proposal{
			Id:          cosmosProp.ProposalID,
			Title:       cosmosProp.Content.Title,
			Description: cosmosProp.Content.Description,
			VotedYes:    yes,
			VotedNo:     no,
			Veto:        veto,
			DeadlineHrs: endsInHrs,
			Voted:       voted,
		})
	}
	return proposals, nil
}

func (cv *CosmosVoter) HasVoted(ctx context.Context, id string) (bool, error) {
	args := strings.Fields(fmt.Sprintf(cosmosHasVotedCmdArgs, id, cv.voterWallet))
	stdout, stderr, err := cv.runner.Run(
		ctx,
		cv.daemonPath,
		args,
		nil,
	)
	if err != nil {
		return false, nil
	}
	hasVoted := cosmosHasVotedResponse{}
	if err := json.Unmarshal(stdout, &hasVoted); err != nil {
		logCmdErr(cv.daemonPath, args, stdout, stderr, err)
		return false, fmt.Errorf("failed to unmarshal voted query response: %v", err)
	}
	return (hasVoted.Option == "VOTE_OPTION_YES"), nil
}

func (cv *CosmosVoter) Vote(ctx context.Context, id string, vote string) error {
	args := strings.Fields(fmt.Sprintf(
		cosmosVoteCmdArgs, id, vote, cv.voterWallet, cv.fees, cv.chainId))
	stdout, stderr, err := cv.runner.Run(
		ctx,
		cv.daemonPath,
		args,
		[]byte(cv.keychainPass),
	)
	if err != nil {
		logCmdErr(cv.daemonPath, args, stdout, stderr, err)
		return fmt.Errorf("failed to run vote tx: %v", err)
	}
	log.Infof("vote tx:\n%s", string(stdout))
	return nil
}

func (cv *CosmosVoter) tally(ctx context.Context, id string) (*cosmosTallyResponse, error) {
	args := strings.Fields(fmt.Sprintf(cosmosTallyCmdArgs, id))
	stdout, stderr, err := cv.runner.Run(
		ctx,
		cv.daemonPath,
		args,
		nil,
	)
	if err != nil {
		logCmdErr(cv.daemonPath, args, stdout, stderr, err)
		return nil, fmt.Errorf("failed to run tally query: %v", err)
	}
	tally := &cosmosTallyResponse{}
	if err := json.Unmarshal(stdout, tally); err != nil {
		logCmdErr(cv.daemonPath, args, stdout, stderr, err)
		return nil, fmt.Errorf("failed to unmarshal tally query response: %v", err)
	}
	return tally, nil
}

func (cv *CosmosVoter) numVoted(ctx context.Context, id string) (int, error) {
	args := strings.Fields(fmt.Sprintf(cosmosNumVotesCmdArgs, id))
	stdout, stderr, err := cv.runner.Run(
		ctx,
		cv.daemonPath,
		args,
		nil,
	)
	if err != nil {
		logCmdErr(cv.daemonPath, args, stdout, stderr, err)
		return 0, fmt.Errorf("failed to run tally query: %v", err)
	}
	votes := &cosmosNumVotesResponse{}
	if err := json.Unmarshal(stdout, votes); err != nil {
		logCmdErr(cv.daemonPath, args, stdout, stderr, err)
		return 0, fmt.Errorf("failed to unmarshal tally query response: %v", err)
	}
	return len(votes.Votes), nil
}

func logCmdErr(cmd string, args []string, stdout []byte, stderr []byte, err error) {
	log.Errorf(
		"Command %s with args %v failed\nCaptured stdout:\n%s\nCaptured stderr:\n%s\n",
		cmd, args, string(stdout), string(stderr),
	)
}
