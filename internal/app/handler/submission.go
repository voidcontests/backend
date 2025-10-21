package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/voidcontests/api/internal/app/handler/dto/request"
	"github.com/voidcontests/api/internal/app/handler/dto/response"
	"github.com/voidcontests/api/internal/lib/logger/sl"
	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/internal/storage/models/status"
	"github.com/voidcontests/api/internal/storage/models/verdict"
	"github.com/voidcontests/api/pkg/requestid"
	"github.com/voidcontests/api/pkg/validate"
)

func (h *Handler) CreateSubmission(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.CreateSubmission"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	contestID, ok := ExtractParamInt(c, "cid")
	if !ok {
		return Error(http.StatusBadRequest, "contest ID should be an integer")
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
	if errors.Is(err, pgx.ErrNoRows) {
		return Error(http.StatusNotFound, "contest not found")
	}
	if err != nil {
		log.Error("can't get contest", sl.Err(err))
		return err
	}

	if contest.StartTime.After(time.Now()) {
		return Error(http.StatusForbidden, "contest is not started yet")
	}

	// TODO: maybe allow to submit solutions after end time if `contest.keep_as_training` is enabled
	if contest.EndTime.Before(time.Now()) {
		return Error(http.StatusForbidden, "contest alreay ended")
	}

	entry, err := h.repo.Entry.Get(ctx, int32(contestID), claims.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		log.Debug("trying to create submission without entry")
		return Error(http.StatusForbidden, "no entry for contest")
	}
	if err != nil {
		log.Error("can't get entry", sl.Err(err))
		return err
	}

	problem, err := h.repo.Problem.Get(ctx, int32(contestID), charcode)
	if errors.Is(err, pgx.ErrNoRows) {
		return Error(http.StatusNotFound, "problem not found")
	}
	if err != nil {
		log.Error("can't get problem", sl.Err(err))
		return err
	}

	if body.ProblemKind == models.TextAnswerProblem {
		var v string
		if problem.Answer != body.Answer {
			v = verdict.WA
		} else {
			v = verdict.OK
		}

		s, err := h.repo.Submission.CreateWithTextAnswer(ctx, entry.ID, problem.ID, v, body.Answer)
		if err != nil {
			log.Error("can't create submission", sl.Err(err))
			return err
		}

		return c.JSON(http.StatusCreated, response.Submission{
			ID:          s.ID,
			ProblemID:   s.ProblemID,
			ProblemKind: s.ProblemKind,
			Status:      s.Status,
			Verdict:     s.Verdict,
			Answer:      s.Answer,
			CreatedAt:   s.CreatedAt,
		})
	} else if body.ProblemKind == models.CodingProblem {
		s, err := h.repo.Submission.CreateWithSolution(ctx, entry.ID, problem.ID, body.Code, body.Language)
		if err != nil {
			log.Error("can't create submission", sl.Err(err))
			return err
		}

		if err := h.broker.PublishSubmission(ctx, s); err != nil {
			log.Error("can't publish submission", sl.Err(err))
			// TODO: if we can't push submission into execution queue, try to save it to local memory, and try to push later (?)
			//   - but is it really needed, after some time?
			if err = h.repo.Submission.UpdateVerdictStatus(ctx, s.ID, verdict.IE, status.Completed); err != nil {
				slog.Error("failed to update submission's verdict", sl.Err(err))
			}
			return err
		}

		return c.JSON(http.StatusCreated, response.Submission{
			ID:          s.ID,
			ProblemID:   s.ProblemID,
			ProblemKind: s.ProblemKind,
			Status:      s.Status,
			Verdict:     s.Verdict,
			CreatedAt:   s.CreatedAt,
		})
	}

	return Error(http.StatusBadRequest, "unknown problem kind")
}

func (h *Handler) GetSubmissionByID(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.GetSubmissionByID"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	submissionID, ok := ExtractParamInt(c, "sid")
	if !ok {
		return Error(http.StatusBadRequest, "submission ID should be an integer")
	}

	s, err := h.repo.Submission.GetByID(ctx, claims.UserID, int32(submissionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Error(http.StatusNotFound, "submission not found")
	}
	if err != nil {
		log.Error("can't get submissions", sl.Err(err))
		return err
	}

	if s.ProblemKind == models.TextAnswerProblem {
		return c.JSON(http.StatusOK, response.Submission{
			ID:          s.ID,
			ProblemID:   s.ProblemID,
			ProblemKind: s.ProblemKind,
			Status:      s.Status,
			Verdict:     s.Verdict,
			Answer:      s.Answer,
			CreatedAt:   s.CreatedAt,
		})
	}

	ttc, err := h.repo.Submission.CountTestsForProblem(ctx, s.ProblemID)
	if err != nil {
		log.Error("can't get total tests count", sl.Err(err))
		return err
	}

	// no need to provide testing report yet (no testing report)
	switch s.Status {
	case status.Pending, status.Running:
		return c.JSON(http.StatusOK, response.Submission{
			ID:          s.ID,
			ProblemID:   s.ProblemID,
			ProblemKind: s.ProblemKind,
			Status:      s.Status,
			Verdict:     s.Verdict,
			Code:        s.Code,
			Language:    s.Language,
			CreatedAt:   s.CreatedAt,
		})
	}

	failedTest, err := h.repo.Submission.GetFailedTest(ctx, s.ID)
	// TODO: check if submission.Passed == submission.Total
	if errors.Is(err, pgx.ErrNoRows) {
		return c.JSON(http.StatusOK, response.Submission{
			ID:          s.ID,
			ProblemID:   s.ProblemID,
			ProblemKind: s.ProblemKind,
			Status:      s.Status,
			Verdict:     s.Verdict,
			Code:        s.Code,
			Language:    s.Language,
			TestingReport: &response.TestingReport{
				Passed: int(s.PassedTestsCount),
				Total:  int(ttc),
				Stderr: s.Stderr,
			},
			CreatedAt: s.CreatedAt,
		})
	}
	if err != nil {
		log.Error("can't get submissions", sl.Err(err))
		return err
	}

	return c.JSON(http.StatusOK, response.Submission{
		ID:          s.ID,
		ProblemID:   s.ProblemID,
		ProblemKind: s.ProblemKind,
		Status:      s.Status,
		Verdict:     s.Verdict,
		Code:        s.Code,
		Language:    s.Language,
		TestingReport: &response.TestingReport{
			Passed: int(s.PassedTestsCount),
			Total:  int(ttc),
			Stderr: s.Stderr,
			FailedTest: &response.FailedTest{
				Input:          failedTest.Input,
				ExpectedOutput: failedTest.ExpectedOutput,
				ActualOutput:   failedTest.ActualOutput,
			},
		},
		CreatedAt: s.CreatedAt,
	})
}

func (h *Handler) GetSubmissions(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.GetSubmissions"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	contestID, ok := ExtractParamInt(c, "cid")
	if !ok {
		return Error(http.StatusBadRequest, "contest ID should be an integer")
	}

	charcode := c.Param("charcode")
	if len(charcode) > 2 {
		return Error(http.StatusBadRequest, "problem's `charcode` couldn't be longer than 2 characters")
	}
	charcode = strings.ToUpper(charcode)

	limit, ok := ExtractQueryParamInt(c, "limit")
	if !ok {
		limit = 10
	}

	offset, ok := ExtractQueryParamInt(c, "offset")
	if !ok {
		offset = 0
	}

	entry, err := h.repo.Entry.Get(ctx, int32(contestID), claims.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Error(http.StatusForbidden, "no entry for contest")
	}
	if err != nil {
		log.Error("can't get entry", sl.Err(err))
		return err
	}

	submissions, total, err := h.repo.Submission.ListByProblem(ctx, entry.ID, charcode, limit, offset)
	if err != nil {
		log.Error("can't get submissions", sl.Err(err))
		return err
	}

	n := len(submissions)
	items := make([]response.Submission, n, n)
	for i, submission := range submissions {
		items[i] = response.Submission{
			ID:          submission.ID,
			ProblemID:   submission.ProblemID,
			ProblemKind: submission.ProblemKind,
			Status:      submission.Status,
			Verdict:     submission.Verdict,
			CreatedAt:   submission.CreatedAt,
		}
	}

	return c.JSON(http.StatusOK, response.Pagination[response.Submission]{
		Meta: response.Meta{
			Total:   total,
			Limit:   limit,
			Offset:  offset,
			HasNext: offset+limit < total,
			HasPrev: offset > 0,
		},
		Items: items,
	})
}
