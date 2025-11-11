package service

import (
	"github.com/voidcontests/api/internal/config"
	"github.com/voidcontests/api/internal/storage/broker"
	"github.com/voidcontests/api/internal/storage/repository"
	"github.com/voidcontests/api/pkg/ton"
)

type Service struct {
	Account    *AccountService
	Submission *SubmissionService
	Problem    *ProblemService
	Contest    *ContestService
	TonProof   *TonProofService
}

func New(cfg *config.Security, repo *repository.Repository, broker broker.Broker, tc *ton.Client) *Service {
	return &Service{
		Account:    NewAccountService(cfg, repo),
		Submission: NewSubmissionService(repo, broker),
		Problem:    NewProblemService(repo),
		Contest:    NewContestService(repo, tc),
		TonProof:   NewTonProofService(repo, tc),
	}
}
