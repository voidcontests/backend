package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/dto/request"
	"github.com/voidcontests/backend/internal/app/handler/dto/response"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/internal/repository/models"
	"github.com/voidcontests/backend/internal/repository/postgres/submission"
	"github.com/voidcontests/backend/pkg/requestid"
	"github.com/voidcontests/backend/pkg/validate"
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
		var verdict string
		if problem.Answer != body.Answer {
			verdict = submission.VerdictWrongAnswer
		} else {
			verdict = submission.VerdictOK
		}

		s, err := h.repo.Submission.Create(ctx, entry.ID, problem.ID, verdict, body.Answer, "", "", 0, "")
		if err != nil {
			log.Error("can't create submission", sl.Err(err))
			return err
		}

		return c.JSON(http.StatusCreated, response.Submission{
			ID:          s.ID,
			ProblemID:   s.ProblemID,
			ProblemKind: s.ProblemKind,
			Verdict:     string(s.Verdict),
			Answer:      body.Answer,
			CreatedAt:   s.CreatedAt,
		})
	} else if body.ProblemKind == models.CodingProblem {
		tcs, err := h.repo.Problem.GetTestCases(ctx, problem.ID)
		if err != nil {
			log.Error("can't get test cases for problem", sl.Err(err))
			return err
		}

		rtcs := make([]request.TC, len(tcs))
		for i := range rtcs {
			rtcs[i].Input = tcs[i].Input
			rtcs[i].Output = tcs[i].Output
		}

		s, err := h.repo.Submission.Create(ctx, entry.ID, problem.ID, submission.VerdictPending, "", body.Code, body.Language, 0, "")
		if err != nil {
			log.Error("can't create submission", sl.Err(err))
			return err
		}

		return c.JSON(http.StatusCreated, response.Submission{
			ID:          s.ID,
			ProblemID:   s.ProblemID,
			ProblemKind: s.ProblemKind,
			Verdict:     submission.VerdictPending,
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

	switch s.Verdict {
	case submission.VerdictRunning, submission.VerdictPending:
		return c.JSON(http.StatusOK, response.Submission{
			ID:          s.ID,
			ProblemID:   s.ProblemID,
			ProblemKind: s.ProblemKind,
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
