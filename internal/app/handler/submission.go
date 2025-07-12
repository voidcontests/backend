package handler

import (
	"database/sql"
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
	"github.com/voidcontests/backend/internal/repository/models"
	"github.com/voidcontests/backend/internal/repository/postgres/submission"
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
	if errors.Is(err, sql.ErrNoRows) {
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
	if errors.Is(err, sql.ErrNoRows) {
		log.Debug("trying to create submission without entry")
		return Error(http.StatusForbidden, "no entry for contest")
	}
	if err != nil {
		log.Error("can't get entry", sl.Err(err))
		return err
	}

	problem, err := h.repo.Problem.Get(ctx, int32(contestID), charcode)
	if errors.Is(err, sql.ErrNoRows) {
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

		submission, err := h.repo.Submission.Create(ctx, entry.ID, problem.ID, verdict, body.Answer, "", "", 0, "")
		if err != nil {
			log.Error("can't create submission", sl.Err(err))
			return err
		}

		return c.JSON(http.StatusCreated, response.Submission{
			ID:        submission.ID,
			ProblemID: submission.ProblemID,
			Verdict:   string(submission.Verdict),
			Answer:    body.Answer,
			CreatedAt: submission.CreatedAt,
		})
	} else if body.ProblemKind == models.CodingProblem {
		tcs, err := h.repo.Problem.GetTCs(ctx, problem.ID)
		if err != nil {
			log.Error("can't get test cases for problem", sl.Err(err))
			return err
		}

		rtcs := make([]request.TC, len(tcs))
		for i := range rtcs {
			rtcs[i].Input = tcs[i].Input
			rtcs[i].Output = tcs[i].Output
		}

		submission, err := h.repo.Submission.Create(ctx, entry.ID, problem.ID, submission.VerdictPending, "", body.Code, body.Language, 0, "")
		if err != nil {
			log.Error("can't create submission", sl.Err(err))
			return err
		}

		return c.JSON(http.StatusCreated, response.ID{
			ID: submission.ID,
		})
	}

	return Error(http.StatusBadRequest, "unknown problem kind")
}

func (h *Handler) GetSubmissionByID(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.GetSubmissionByID"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	sid := c.Param("sid")
	submissionID, err := strconv.Atoi(sid)
	if err != nil {
		log.Debug("`sid` param is not an integer", slog.String("sid", sid), sl.Err(err))
		return Error(http.StatusBadRequest, "`sid` should be integer")
	}

	submission, err := h.repo.Submission.GetByID(ctx, claims.UserID, int32(submissionID))
	if errors.Is(err, sql.ErrNoRows) {
		return Error(http.StatusNotFound, "submission not found")
	}
	if err != nil {
		log.Error("can't get submissions", sl.Err(err))
		return err
	}

	ttc, err := h.repo.Submission.GetTotalTestsCount(ctx, submission.ProblemID)
	if err != nil {
		log.Error("can't get total tests count", sl.Err(err))
		return err
	}

	failedTest, err := h.repo.Submission.GetFailedTest(ctx, submission.ID)
	// TODO: check if submission.Passed == submission.Total
	if errors.Is(err, sql.ErrNoRows) {
		return c.JSON(http.StatusCreated, response.Submission{
			ID:        submission.ID,
			ProblemID: submission.ProblemID,
			Verdict:   submission.Verdict,
			Code:      submission.Code,
			Language:  submission.Language,
			TestingReport: response.TestingReport{
				Passed: int(submission.PassedTestsCount),
				Total:  int(ttc),
			},
			CreatedAt: submission.CreatedAt,
		})
	}
	if err != nil {
		log.Error("can't get submissions", sl.Err(err))
		return err
	}

	return c.JSON(http.StatusCreated, response.Submission{
		ID:        submission.ID,
		ProblemID: submission.ProblemID,
		Verdict:   submission.Verdict,
		Code:      submission.Code,
		Language:  submission.Language,
		TestingReport: response.TestingReport{
			Passed: int(submission.PassedTestsCount),
			Total:  int(ttc),
			Stderr: submission.Stderr,
			FailedTest: &response.FailedTest{
				Input:          failedTest.Input,
				ExpectedOutput: failedTest.ExpectedOutput,
				ActualOutput:   failedTest.ActualOutput,
			},
		},
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

	entry, err := h.repo.Entry.Get(ctx, int32(contestID), claims.UserID)
	if errors.Is(err, sql.ErrNoRows) {
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
	ss := make([]response.Submission, n, n)
	for i, s := range submissions {
		ss[i] = response.Submission{
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
