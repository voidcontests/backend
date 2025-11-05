package service

import "errors"

var (
	// account
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrTokenGeneration    = errors.New("failed to generate token")
	ErrInvalidToken       = errors.New("invalid or expired token")

	// contest
	ErrUnknownAwardType = errors.New("unknown award type")

	// entry
	ErrContestFinished     = errors.New("contest not found")
	ErrContestNotFound     = errors.New("contest not found")
	ErrMaxSlotsReached     = errors.New("max slots limit reached")
	ErrApplicationTimeOver = errors.New("application time is over")
	ErrEntryAlreadyExists  = errors.New("user already has entry for this contest")
	ErrEntryNotFound       = errors.New("entry not found")

	// problem
	ErrUserBanned            = errors.New("you are banned from creating problems")
	ErrProblemsLimitExceeded = errors.New("problems limit exceeded")
	ErrContestsLimitExceeded = errors.New("contests limit exceeded")
	ErrInvalidTimeLimit      = errors.New("time_limit_ms must be between 500 and 10000")
	ErrInvalidMemoryLimit    = errors.New("memory_limit_mb must be between 16 and 512")
	ErrContestNotStarted     = errors.New("contest not started yet")
	ErrNotProblemWriter      = errors.New("you are not the problem writer")

	// submissions
	ErrProblemNotFound        = errors.New("problem not found")
	ErrNoEntryForContest      = errors.New("no entry for contest")
	ErrSubmissionWindowClosed = errors.New("submission window is currently closed")
	ErrSubmissionNotFound     = errors.New("submission not found")
	ErrInvalidCharcode        = errors.New("problem's charcode couldn't be longer than 2 characters")
)
