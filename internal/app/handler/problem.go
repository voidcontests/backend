package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/api/internal/app/handler/dto/request"
	"github.com/voidcontests/api/internal/app/handler/dto/response"
	"github.com/voidcontests/api/internal/app/service"
	"github.com/voidcontests/api/pkg/validate"
)

func (h *Handler) CreateProblem(c echo.Context) error {
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	var body request.CreateProblemRequest
	if err := validate.Bind(c, &body); err != nil {
		return Error(http.StatusBadRequest, "invalid body: missing required fields")
	}

	id, err := h.service.Problem.CreateProblem(ctx, claims.UserID, body.Title, body.Statement, body.Difficulty, body.TimeLimitMS, body.MemoryLimitMB, body.Checker, body.TestCases)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserBanned):
			return Error(http.StatusForbidden, "you are banned from creating problems")
		case errors.Is(err, service.ErrProblemsLimitExceeded):
			return Error(http.StatusForbidden, "problems limit exceeded")
		case errors.Is(err, service.ErrInvalidTimeLimit):
			return Error(http.StatusBadRequest, "time_limit_ms must be between 500 and 10000")
		case errors.Is(err, service.ErrInvalidMemoryLimit):
			return Error(http.StatusBadRequest, "memory_limit_mb must be between 16 and 512")
		default:
			return err
		}
	}

	return c.JSON(http.StatusCreated, response.ID{
		ID: id,
	})
}

func (h *Handler) GetCreatedProblems(c echo.Context) error {
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	limit, ok := ExtractQueryParamInt(c, "limit")
	if !ok {
		limit = 10
	}

	offset, ok := ExtractQueryParamInt(c, "offset")
	if !ok {
		offset = 0
	}

	result, err := h.service.Problem.GetCreatedProblems(ctx, claims.UserID, limit, offset)
	if err != nil {
		return err
	}

	n := len(result.Problems)
	problems := make([]response.ProblemListItem, n, n)
	for i, p := range result.Problems {
		problems[i] = response.ProblemListItem{
			ID:            p.ID,
			Title:         p.Title,
			Difficulty:    p.Difficulty,
			CreatedAt:     p.CreatedAt,
			TimeLimitMS:   p.TimeLimitMS,
			MemoryLimitMB: p.MemoryLimitMB,
			Checker:       p.Checker,
			Writer: response.User{
				ID:       p.WriterID,
				Username: p.WriterUsername,
			},
		}
	}

	return c.JSON(http.StatusOK, response.Pagination[response.ProblemListItem]{
		Meta: response.Meta{
			Total:   result.Total,
			Limit:   limit,
			Offset:  offset,
			HasNext: offset+limit < result.Total,
			HasPrev: offset > 0,
		},
		Items: problems,
	})
}

func (h *Handler) GetContestProblem(c echo.Context) error {
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	contestID, ok := ExtractParamInt(c, "cid")
	if !ok {
		return Error(http.StatusBadRequest, "contest ID should be an integer")
	}

	charcode := c.Param("charcode")

	details, err := h.service.Problem.GetContestProblem(ctx, int32(contestID), claims.UserID, charcode)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCharcode):
			return Error(http.StatusBadRequest, "problem charcode couldn't be longer than 2 characters")
		case errors.Is(err, service.ErrContestNotFound):
			return Error(http.StatusNotFound, "contest not found")
		case errors.Is(err, service.ErrContestNotStarted):
			return Error(http.StatusForbidden, "contest not started yet")
		case errors.Is(err, service.ErrNoEntryForContest):
			return Error(http.StatusForbidden, "no entry")
		case errors.Is(err, service.ErrProblemNotFound):
			return Error(http.StatusNotFound, "problem not found")
		default:
			return err
		}
	}

	p := details.Problem
	n := len(details.Examples)
	examples := make([]response.TC, n, n)
	for i := 0; i < n; i++ {
		examples[i] = response.TC{
			Input:  details.Examples[i].Input,
			Output: details.Examples[i].Output,
		}
	}

	pdetailed := response.ContestProblemDetailed{
		ID:            p.ID,
		Charcode:      p.Charcode,
		ContestID:     int32(contestID),
		Title:         p.Title,
		Statement:     p.Statement,
		Examples:      examples,
		Difficulty:    p.Difficulty,
		Status:        details.Status,
		CreatedAt:     p.CreatedAt,
		TimeLimitMS:   p.TimeLimitMS,
		MemoryLimitMB: p.MemoryLimitMB,
		Checker:       p.Checker,
		Writer: response.User{
			ID:       p.WriterID,
			Username: p.WriterUsername,
		},
	}

	if details.SubmissionWindow.Earliest.Before(time.Now()) {
		pdetailed.SubmissionDeadline = &details.SubmissionWindow.Deadline
	}

	return c.JSON(http.StatusOK, pdetailed)
}

func (h *Handler) GetProblemByID(c echo.Context) error {
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	problemID, ok := ExtractParamInt(c, "pid")
	if !ok {
		return Error(http.StatusBadRequest, "problem ID should be an integer")
	}

	details, err := h.service.Problem.GetProblemByID(ctx, int32(problemID), claims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrProblemNotFound):
			return Error(http.StatusNotFound, "problem not found")
		case errors.Is(err, service.ErrNotProblemWriter):
			return Error(http.StatusNotFound, "problem not found")
		default:
			return err
		}
	}

	problem := details.Problem
	n := len(details.Examples)
	examples := make([]response.TC, n, n)
	for i := 0; i < n; i++ {
		examples[i] = response.TC{
			Input:  details.Examples[i].Input,
			Output: details.Examples[i].Output,
		}
	}

	pdetailed := response.ProblemDetailed{
		ID:            problem.ID,
		Title:         problem.Title,
		Statement:     problem.Statement,
		Examples:      examples,
		Difficulty:    problem.Difficulty,
		CreatedAt:     problem.CreatedAt,
		TimeLimitMS:   problem.TimeLimitMS,
		MemoryLimitMB: problem.MemoryLimitMB,
		Checker:       problem.Checker,
		Writer: response.User{
			ID:       problem.WriterID,
			Username: problem.WriterUsername,
		},
	}

	return c.JSON(http.StatusOK, pdetailed)
}
