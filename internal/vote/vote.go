package vote

import (
	"context"
)

type Proposal struct {
	Id          string
	Title       string
	Description string
	VotedYes    int
	VotedNo     int
	Veto        int
}

//go:generate mockgen -source vote.go -destination vote_mock.go -package vote
type Voter interface {
	GetVoting(context.Context) ([]Proposal, error)
	HasVoted(context.Context, string) (bool, error)
	Vote(context.Context, string, bool) error
}
