package errors

import "errors"

var (
	ErrContestNotFound = errors.New("repo: contest not found")
	ErrProblemNotFound = errors.New("repo: problem not found")
)
