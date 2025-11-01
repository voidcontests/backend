package service

import (
	"github.com/voidcontests/api/internal/config"
	"github.com/voidcontests/api/internal/storage/broker"
	"github.com/voidcontests/api/internal/storage/repository"
)

type Service struct {
	Account    *AccountService
	Entry      *EntryService
	Submission *SubmissionService
	Problem    *ProblemService
	Contest    *ContestService
}

func New(cfg *config.Config, repo *repository.Repository, broker broker.Broker) *Service {
	return &Service{
		Account:    NewAccountService(cfg, repo),
		Entry:      NewEntryService(repo),
		Submission: NewSubmissionService(repo, broker),
		Problem:    NewProblemService(repo),
		Contest:    NewContestService(repo),
	}
}
