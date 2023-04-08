package vote

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/kostage/cosmos_voter/internal/cmdrunner"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var (
	cosmosGetVotingCmdArgs  = "query gov proposals --status VotingPeriod -o json"
	cosmosHasVotedCmdArgs   = "query gov vote %s %s -o json"
	cosmosTallyCmdArgs      = "query gov tally %s -o json"
	cosmosVoteCmdArgs       = "tx gov vote %s %s --from %s --fees %s --chain-id %s -y"
	cosmosValidatorsCmdArgs = "query tendermint-validator-set"

	defRunnerFactory = cmdrunner.NewCmdRunner
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
	Option  string              `json:"option"`
	Options []cosmosVotedOption `json:"options"`
}

type cosmosVotedOption struct {
	Option string `json:"option"`
}

type cosmosTallyResponse struct {
	Yes        int `json:"yes,string"`
	Abstain    int `json:"abstain,string"`
	No         int `json:"no,string"`
	NoWithVeto int `json:"no_with_veto,string"`
}

type cosmosNumVotesResponse struct {
	Votes []interface{} `json:"votes"`
}

type cosmosValidatorsResponse struct {
	Validators []cosmosValidator `yaml:"validators"`
}

type cosmosValidator struct {
	VotingPower string `yaml:"voting_power"`
}

type CosmosVoter struct {
	daemonPath   string
	keychainPass string
	voterWallet  string
	fees         string
	chainId      string
}

func NewCosmosVoter(
	daemonPath string,
	keychainPass string,
	voterWallet string,
	fees string,
	chainId string,
) *CosmosVoter {
	return &CosmosVoter{
		daemonPath:   daemonPath,
		keychainPass: keychainPass,
		voterWallet:  voterWallet,
		fees:         fees,
		chainId:      chainId,
	}
}

func (cv *CosmosVoter) GetVoting(ctx context.Context) ([]Proposal, error) {
	args := strings.Fields(cosmosGetVotingCmdArgs)
	runner := defRunnerFactory()
	stdout, stderr, err := runner.Run(
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
	totalPower, err := cv.totalVotingPower(ctx)
	if err != nil {
		return nil, err
	}
	proposals := make([]Proposal, 0, len(cosmosProposals.Proposals))
	for _, cosmosProp := range cosmosProposals.Proposals {
		tally, err := cv.tally(ctx, cosmosProp.ProposalID)
		if err != nil {
			return nil, err
		}
		all := float64(tally.Yes + tally.No + tally.NoWithVeto + tally.Abstain)
		yes := float64(tally.Yes) * 100 / all
		no := float64(tally.No) * 100 / all
		veto := float64(tally.NoWithVeto) * 100 / all
		endsInHrs := cosmosProp.VotingEndTime.Sub(time.Now().UTC()).Hours()
		endsInHrs = math.Round(endsInHrs*100) / 100
		voted := float64(all) / float64(totalPower*10000)
		voted = math.Round(voted*100) / 100
		proposals = append(proposals, Proposal{
			Id:          cosmosProp.ProposalID,
			Title:       cosmosProp.Content.Title,
			Description: cosmosProp.Content.Description,
			VotedYes:    math.Round(yes*100) / 100,
			VotedNo:     math.Round(no*100) / 100,
			Veto:        math.Round(veto*100) / 100,
			DeadlineHrs: endsInHrs,
			Voted:       voted,
		})
	}
	return proposals, nil
}

func (cv *CosmosVoter) HasVoted(ctx context.Context, id string) (bool, error) {
	args := strings.Fields(fmt.Sprintf(cosmosHasVotedCmdArgs, id, cv.voterWallet))
	runner := defRunnerFactory()
	stdout, stderr, err := runner.Run(
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
	return (len(hasVoted.Options) > 0 && hasVoted.Options[0].Option == "VOTE_OPTION_YES"), nil
}

func (cv *CosmosVoter) Vote(ctx context.Context, id string, vote string) error {
	args := strings.Fields(fmt.Sprintf(
		cosmosVoteCmdArgs, id, vote, cv.voterWallet, cv.fees, cv.chainId))
	runner := defRunnerFactory()
	stdout, stderr, err := runner.Run(
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
	runner := defRunnerFactory()
	stdout, stderr, err := runner.Run(
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

func (cv *CosmosVoter) totalVotingPower(ctx context.Context) (int, error) {
	args := strings.Fields(fmt.Sprintf(cosmosValidatorsCmdArgs))
	runner := defRunnerFactory()
	stdout, stderr, err := runner.Run(
		ctx,
		cv.daemonPath,
		args,
		nil,
	)
	if err != nil {
		logCmdErr(cv.daemonPath, args, stdout, stderr, err)
		return 0, fmt.Errorf("failed to run tally query: %v", err)
	}
	validators := &cosmosValidatorsResponse{}
	if err := yaml.Unmarshal(stdout, validators); err != nil {
		logCmdErr(cv.daemonPath, args, stdout, stderr, err)
		return 0, fmt.Errorf("failed to unmarshal tendermint validators response: %v", err)
	}
	totalPower := 0
	for _, validator := range validators.Validators {
		pow, err := strconv.Atoi(validator.VotingPower)
		if err != nil {
			return 0, fmt.Errorf("failed to unmarshal tendermint validators response: voting power is not integer")
		}
		totalPower += pow
	}
	return totalPower, nil
}

func logCmdErr(cmd string, args []string, stdout []byte, stderr []byte, err error) {
	log.Errorf(
		"Command %s with args %v failed\nCaptured stdout:\n%s\nCaptured stderr:\n%s\n",
		cmd, args, string(stdout), string(stderr),
	)
}
