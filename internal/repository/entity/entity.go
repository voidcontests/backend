package entity

import "time"

type Contest struct {
	ID             int32
	Title          string
	Description    string
	ProblemIDs     []int32
	CreatorAddress string
	Start          time.Time
	Duration       time.Duration
	Slots          int32
	IsDraft        bool
	CreatedAt      time.Time
}

type Problem struct {
	ID            int32
	Title         string
	Task          string
	WriterAddress string
	Input         string
	Answer        string
	CreatedAt     time.Time
}
