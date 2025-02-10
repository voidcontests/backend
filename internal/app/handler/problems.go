package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/dto/request"
	"github.com/voidcontests/backend/internal/app/handler/dto/response"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/internal/repository/postgres/submission"
	"github.com/voidcontests/backend/internal/repository/repoerr"
	"github.com/voidcontests/backend/pkg/requestid"
	"github.com/voidcontests/backend/pkg/validate"
)

func (h *Handler) CreateProblem(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.CreateProblem"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	var body request.CreateProblemRequest
	if err := validate.Bind(c, &body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		return Error(http.StatusBadRequest, "invalid body: missing required fields")
	}

	problemID, err := h.repo.Problem.Create(ctx, claims.ID, body.Title, body.Statement, body.Difficulty, body.Input, body.Answer)
	if err != nil {
		log.Error("can't create problem", sl.Err(err))
		return err
	}

	return c.JSON(http.StatusCreated, response.ContestID{
		ID: problemID,
	})
}

func (h *Handler) GetProblems(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.GetProblems"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	// TODO: return problems splitted by chunks
	ps, err := h.repo.Problem.GetAll(ctx)
	if err != nil {
		log.Error("can't get contests", sl.Err(err))
		return err
	}

	n := len(ps)
	problems := make([]response.ProblemListItem, n, n)
	for i, p := range ps {
		problems[i] = response.ProblemListItem{
			ID:         p.ID,
			Title:      p.Title,
			Difficulty: p.Difficulty,
			Writer: response.User{
				ID:      p.WriterID,
				Address: p.WriterAddress,
			},
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": problems,
	})
}

func (h *Handler) GetProblem(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.GetProblem"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	cid := c.Param("cid")
	contestID, err := strconv.Atoi(cid)
	if err != nil {
		log.Debug("`cid` param is not an integer", slog.String("cid", cid), sl.Err(err))
		return Error(http.StatusBadRequest, "`cid` should be integer")
	}

	pid := c.Param("pid")
	problemID, err := strconv.Atoi(pid)
	if err != nil {
		log.Debug("`pid` param is not an integer", slog.String("pid", pid), sl.Err(err))
		return Error(http.StatusBadRequest, "`pid` should be integer")
	}

	entry, err := h.repo.Entry.Get(ctx, int32(contestID), claims.ID)
	if errors.Is(err, repoerr.ErrEntryNotFound) {
		log.Debug("no entry for contest")
		return Error(http.StatusForbidden, "no entry")
	}
	if err != nil {
		log.Debug("can't get entry", sl.Err(err))
		return err
	}

	p, err := h.repo.Problem.Get(ctx, int32(contestID), int32(problemID))
	if errors.Is(err, repoerr.ErrProblemNotFound) {
		return Error(http.StatusNotFound, "problem not found")
	}
	if err != nil {
		log.Debug("can't get problem", sl.Err(err))
		return err
	}

	submissions, err := h.repo.Submission.GetForProblem(ctx, entry.ID, p.ID)
	if err != nil {
		log.Error("can't get submissions", sl.Err(err))
		return err
	}

	pdetailed := response.ProblemDetailed{
		ID:         p.ID,
		ContestID:  int32(contestID),
		Title:      p.Title,
		Statement:  p.Statement,
		Difficulty: p.Difficulty,
		Input:      p.Input,
		Writer: response.User{
			ID:      p.WriterID,
			Address: p.WriterAddress,
		},
	}

	// TODO: Make status enum
	for i := 0; i < len(submissions) && pdetailed.Status != "accepted"; i++ {
		switch submissions[i].Verdict {
		case submission.VerdictOK:
			pdetailed.Status = "accepted"
		case submission.VerdictWrongAnswer:
			pdetailed.Status = "tried"
		}
	}

	return c.JSON(http.StatusOK, pdetailed)
}
