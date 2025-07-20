package service

import "errors"

var (
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
	ErrUserBanned        = errors.New("banned")

	ErrContestNotFound       = errors.New("contest not found")
	ErrEntriesLimitReached   = errors.New("max slots limit reached")
	ErrApplicationTimeIsOver = errors.New("application time is over")
	ErrEntryAlreadyExists    = errors.New("user already has entry for this contest")

	ErrContestsLimitExceeded = errors.New("contests limit exceeded")
)
