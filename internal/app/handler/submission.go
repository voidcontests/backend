package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/dto/request"
	"github.com/voidcontests/backend/internal/app/handler/dto/response"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/internal/repository/postgres/submission"
	"github.com/voidcontests/backend/internal/repository/repoerr"
	"github.com/voidcontests/backend/pkg/requestid"
	"github.com/voidcontests/backend/pkg/validate"
)

func (h *Handler) CreateSubmission(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.CreateSubmission"), slog.String("request_id", requestid.Get(c)))
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

	var body request.CreateSubmissionRequest
	if err := validate.Bind(c, &body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		return Error(http.StatusBadRequest, "invalid body")
	}

	contest, err := h.repo.Contest.GetByID(ctx, int32(contestID))
	if errors.Is(err, repoerr.ErrContestNotFound) {
		return Error(http.StatusNotFound, "contest not found")
	}
	if err != nil {
		log.Error("can't get contest", sl.Err(err))
		return err
	}

	if contest.StartTime.After(time.Now()) {
		return Error(http.StatusForbidden, "contest is not started yet")
	}

	// TODO: maybe allow to submit solutions after end time if `keep_as_training` is enabled
	if contest.EndTime.Before(time.Now()) {
		return Error(http.StatusForbidden, "contest alreay ended")
	}

	entry, err := h.repo.Entry.Get(ctx, int32(contestID), claims.ID)
	if errors.Is(err, repoerr.ErrEntryNotFound) {
		log.Debug("trying to create submission without entry")
		return Error(http.StatusForbidden, "no entry for contest")
	}
	if err != nil {
		log.Error("can't get entry", sl.Err(err))
		return err
	}

	problem, err := h.repo.Problem.Get(ctx, int32(contestID), charcode)
	if errors.Is(err, repoerr.ErrProblemNotFound) {
		return Error(http.StatusNotFound, "problem not found")
	}
	if err != nil {
		log.Error("can't get problem", sl.Err(err))
		return err
	}

	var verdict string
	if problem.Answer != body.Answer {
		verdict = submission.VerdictWrongAnswer
	} else {
		verdict = submission.VerdictOK
	}

	submission, err := h.repo.Submission.Create(ctx, entry.ID, problem.ID, verdict, body.Answer)
	if err != nil {
		log.Error("can't create submission", sl.Err(err))
		return err
	}

	return c.JSON(http.StatusCreated, response.SubmissionListItem{
		ID:        submission.ID,
		ProblemID: submission.ProblemID,
		Verdict:   string(submission.Verdict),
		CreatedAt: submission.CreatedAt,
	})
}

func (h *Handler) GetSubmissions(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.GetSubmissions"), slog.String("request_id", requestid.Get(c)))
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
		return Error(http.StatusForbidden, "no entry for contest")
	}
	if err != nil {
		log.Error("can't get entry", sl.Err(err))
		return err
	}

	submissions, err := h.repo.Submission.GetForProblem(ctx, entry.ID, charcode)
	if err != nil {
		log.Error("can't get submissions", sl.Err(err))
		return err
	}

	n := len(submissions)
	ss := make([]response.SubmissionListItem, n, n)
	for i, s := range submissions {
		ss[i] = response.SubmissionListItem{
			ID:        s.ID,
			ProblemID: s.ProblemID,
			Verdict:   string(s.Verdict),
			CreatedAt: s.CreatedAt,
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": ss,
	})
}
