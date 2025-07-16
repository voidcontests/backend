package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/dto/request"
	"github.com/voidcontests/backend/internal/app/handler/dto/response"
	"github.com/voidcontests/backend/internal/repository/models"
	"github.com/voidcontests/backend/pkg/validate"
)

func (h *Handler) CreateProblem(c echo.Context) error {
	op := "handler.CreateProblem"
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	var body request.CreateProblemRequest
	if err := validate.Bind(c, &body); err != nil {
		return Error(http.StatusBadRequest, "invalid body: missing required fields")
	}

	userrole, err := h.repo.User.GetRole(ctx, claims.UserID)
	if err != nil {
		return fmt.Errorf("%s: can't get role: %v", op, err)
	}

	if userrole.Name == models.RoleBanned {
		return Error(http.StatusForbidden, "you are banned from creating problems")
	}

	if userrole.Name == models.RoleLimited {
		pscount, err := h.repo.User.GetCreatedProblemsCount(ctx, claims.UserID)
		if err != nil {
			return fmt.Errorf("%s: can't get created problems count: %v", op, err)
		}

		if pscount >= int(userrole.CreatedProblemsLimit) {
			return Error(http.StatusForbidden, "problems limit exceeded")
		}
	}

	var problemID int32
	if body.Kind == models.TextAnswerProblem {
		problemID, err = h.repo.Problem.Create(ctx, models.TextAnswerProblem, claims.UserID, body.Title, body.Statement, body.Difficulty, body.Answer, 0)
	} else if body.Kind == models.CodingProblem {
		examplesCount := 0
		for i := range body.TestCases {
			if body.TestCases[i].IsExample {
				examplesCount++
			}

			if examplesCount > 3 && body.TestCases[i].IsExample {
				body.TestCases[i].IsExample = false
			}
		}
		problemID, err = h.repo.Problem.CreateWithTCs(ctx, models.CodingProblem, claims.UserID, body.Title, body.Statement, body.Difficulty, "", body.TimeLimitMS, body.TestCases)
	} else {
		return Error(http.StatusBadRequest, "unknown problem kind")
	}

	if err != nil {
		return fmt.Errorf("%s: can't create problem: %v", op, err)
	}

	return c.JSON(http.StatusCreated, response.ID{
		ID: problemID,
	})
}

func (h *Handler) GetCreatedProblems(c echo.Context) error {
	op := "handler.GetCreatedProblems"
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

	ps, total, err := h.repo.Problem.GetWithWriterID(ctx, claims.UserID, limit, offset)
	if err != nil {
		return fmt.Errorf("%s: can't get created contests: %v", op, err)
	}

	n := len(ps)
	problems := make([]response.ProblemListItem, n, n)
	for i, p := range ps {
		problems[i] = response.ProblemListItem{
			ID:         p.ID,
			Title:      p.Title,
			Difficulty: p.Difficulty,
			CreatedAt:  p.CreatedAt,
			Writer: response.User{
				ID:       p.WriterID,
				Username: p.WriterUsername,
			},
		}
	}

	return c.JSON(http.StatusOK, response.Pagination[response.ProblemListItem]{
		Meta: response.Meta{
			Total:   total,
			Limit:   limit,
			Offset:  offset,
			HasNext: offset+limit < total,
			HasPrev: offset > 0,
		},
		Items: problems,
	})
}

func (h *Handler) GetContestProblem(c echo.Context) error {
	op := "handler.GetContestProblem"
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	contestID, ok := ExtractParamInt(c, "cid")
	if !ok {
		return Error(http.StatusBadRequest, "contest ID should be an integer")
	}

	charcode := c.Param("charcode")
	if len(charcode) > 2 {
		return Error(http.StatusBadRequest, "problem charcode couldn't be longer than 2 characters")
	}
	charcode = strings.ToUpper(charcode)

	entry, err := h.repo.Entry.Get(ctx, int32(contestID), claims.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Error(http.StatusForbidden, "no entry")
	}
	if err != nil {
		return fmt.Errorf("%s: can't get entry: %v", op, err)
	}

	p, err := h.repo.Problem.Get(ctx, int32(contestID), charcode)
	if errors.Is(err, pgx.ErrNoRows) {
		return Error(http.StatusNotFound, "problem not found")
	}
	if err != nil {
		return fmt.Errorf("%s: can't get problem: %v", op, err)
	}

	etc, err := h.repo.Problem.GetExampleCases(ctx, p.ID)
	if err != nil {
		return fmt.Errorf("%s: can't get tc examples: %v", op, err)
	}

	n := len(etc)
	examples := make([]response.TC, n, n)
	for i := 0; i < n; i++ {
		examples[i] = response.TC{
			Input:  etc[i].Input,
			Output: etc[i].Output,
		}
	}

	status, err := h.repo.Submission.GetProblemStatus(ctx, entry.ID, p.ID)
	if err != nil {
		return err
	}

	pdetailed := response.ProblemDetailed{
		ID:          p.ID,
		Charcode:    p.Charcode,
		ContestID:   int32(contestID),
		Kind:        p.Kind,
		Title:       p.Title,
		Statement:   p.Statement,
		Examples:    examples,
		Difficulty:  p.Difficulty,
		Status:      status,
		CreatedAt:   p.CreatedAt,
		TimeLimitMS: p.TimeLimitMS,
		Writer: response.User{
			ID:       p.WriterID,
			Username: p.WriterUsername,
		},
	}

	return c.JSON(http.StatusOK, pdetailed)
}
