package service

import (
	"github.com/voidcontests/api/internal/config"
	"github.com/voidcontests/api/internal/storage/broker"
	"github.com/voidcontests/api/internal/storage/repository"
	"github.com/voidcontests/api/pkg/ton"
)

type Service struct {
	Account    *AccountService
	Entry      *EntryService
	Submission *SubmissionService
	Problem    *ProblemService
	Contest    *ContestService
}

func New(cfg *config.Security, repo *repository.Repository, broker broker.Broker, tc *ton.Client) *Service {
	return &Service{
		// TODO: pass only salt and signature key, not entire config
		Account:    NewAccountService(cfg, repo),
		Entry:      NewEntryService(repo),
		Submission: NewSubmissionService(repo, broker),
		Problem:    NewProblemService(repo),
		Contest:    NewContestService(repo, tc),
	}
}
