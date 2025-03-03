package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/dto/request"
	"github.com/voidcontests/backend/internal/app/handler/dto/response"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/internal/repository/models"
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

	userrole, err := h.repo.User.GetRole(ctx, claims.ID)
	if err != nil {
		log.Error("can't get user's role", sl.Err(err))
		return err
	}

	if userrole.Name == models.RoleBanned {
		log.Debug("banned mf tried to create new problem")
		return Error(http.StatusForbidden, "you are banned from creating problems")
	}

	if userrole.Name == models.RoleLimited {
		pscount, err := h.repo.User.GetCreatedProblemsCount(ctx, claims.ID)
		if err != nil {
			log.Debug("can't get created problems count", sl.Err(err))
			return err
		}

		if pscount >= int(userrole.CreatedProblemsLimit) {
			return Error(http.StatusForbidden, "problems limit exceeded")
		}
	}

	var problemID int32
	if body.Kind == models.TextAnswerProblem {
		problemID, err = h.repo.Problem.Create(ctx, models.TextAnswerProblem, claims.ID, body.Title, body.Statement, body.Difficulty, body.Input, body.Answer, 0)
	} else if body.Kind == models.CodingProblem {
		problemID, err = h.repo.Problem.CreateWithTCs(ctx, models.CodingProblem, claims.ID, body.Title, body.Statement, body.Difficulty, "", "", int32(body.TimeLimitMS), body.TestCases)
	} else {
		log.Debug("unknown problem kind", slog.String("problem_kind", body.Kind))
		return Error(http.StatusBadRequest, "unknown problem kind")
	}

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
			CreatedAt:  p.CreatedAt,
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

func (h *Handler) GetCreatedProblems(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.GetCreatedProblems"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	// TODO: return problems splitted by chunks
	ps, err := h.repo.Problem.GetWithWriterID(ctx, claims.ID)
	if err != nil {
		log.Error("can't get created contests", sl.Err(err))
		return err
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

	charcode := c.Param("charcode")
	if len(charcode) > 2 {
		return Error(http.StatusBadRequest, "problem's `charcode` couldn't be longer than 2 characters")
	}
	charcode = strings.ToUpper(charcode)

	entry, err := h.repo.Entry.Get(ctx, int32(contestID), claims.ID)
	if errors.Is(err, repoerr.ErrEntryNotFound) {
		log.Debug("no entry for contest")
		return Error(http.StatusForbidden, "no entry")
	}
	if err != nil {
		log.Debug("can't get entry", sl.Err(err))
		return err
	}

	p, err := h.repo.Problem.Get(ctx, int32(contestID), charcode)
	if errors.Is(err, repoerr.ErrProblemNotFound) {
		return Error(http.StatusNotFound, "problem not found")
	}
	if err != nil {
		log.Debug("can't get problem", sl.Err(err))
		return err
	}

	submissions, err := h.repo.Submission.GetForProblem(ctx, entry.ID, charcode)
	if err != nil {
		log.Error("can't get submissions", sl.Err(err))
		return err
	}

	pdetailed := response.ProblemDetailed{
		ID:          p.ID,
		Charcode:    p.Charcode,
		ContestID:   int32(contestID),
		Kind:        p.Kind,
		Title:       p.Title,
		Statement:   p.Statement,
		Difficulty:  p.Difficulty,
		Input:       p.Input,
		CreatedAt:   p.CreatedAt,
		TimeLimitMS: p.TimeLimitMS,
		Writer: response.User{
			ID:      p.WriterID,
			Address: p.WriterAddress,
		},
	}

	for i := 0; i < len(submissions) && pdetailed.Status != "accepted"; i++ {
		switch submissions[i].Verdict {
		case submission.VerdictOK:
			pdetailed.Status = "accepted"
		default:
			pdetailed.Status = "tried"
		}
	}

	return c.JSON(http.StatusOK, pdetailed)
}
