package repoerr

import "errors"

var (
	ErrUserNotFound    = errors.New("repo: user not found")
	ErrContestNotFound = errors.New("repo: contest not found")
	ErrProblemNotFound = errors.New("repo: problem not found")
	ErrEntryNotFound   = errors.New("repo: entry not found")
)
