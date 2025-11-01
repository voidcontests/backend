package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/api/internal/app/handler/dto/request"
	"github.com/voidcontests/api/internal/app/handler/dto/response"
	"github.com/voidcontests/api/internal/app/service"
	"github.com/voidcontests/api/internal/lib/logger/sl"
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

	var body request.CreateSubmissionRequest
	if err := validate.Bind(c, &body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		return Error(http.StatusBadRequest, "invalid body")
	}

	result, err := h.service.Submission.CreateSubmission(ctx, int32(contestID), claims.UserID, charcode, body.Code, body.Language)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCharcode):
			return Error(http.StatusBadRequest, "problem's `charcode` couldn't be longer than 2 characters")
		case errors.Is(err, service.ErrContestNotFound):
			return Error(http.StatusNotFound, "contest not found")
		case errors.Is(err, service.ErrNoEntryForContest):
			log.Debug("trying to create submission without entry")
			return Error(http.StatusForbidden, "no entry for contest")
		case errors.Is(err, service.ErrSubmissionWindowClosed):
			return Error(http.StatusForbidden, "submission window is currently closed")
		case errors.Is(err, service.ErrProblemNotFound):
			return Error(http.StatusNotFound, "problem not found")
		default:
			log.Error("failed to create submission", sl.Err(err))
			return err
		}
	}

	s := result.Submission
	return c.JSON(http.StatusCreated, response.Submission{
		ID:        s.ID,
		ProblemID: s.ProblemID,
		Status:    s.Status,
		Verdict:   s.Verdict,
		CreatedAt: s.CreatedAt,
	})
}

func (h *Handler) GetSubmissionByID(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.GetSubmissionByID"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	// TODO: check if submission is submitted by request initiator
	_, _ = ExtractClaims(c)

	submissionID, ok := ExtractParamInt(c, "sid")
	if !ok {
		return Error(http.StatusBadRequest, "submission ID should be an integer")
	}

	details, err := h.service.Submission.GetSubmissionByID(ctx, int32(submissionID))
	if err != nil {
		if errors.Is(err, service.ErrSubmissionNotFound) {
			return Error(http.StatusNotFound, "submission not found")
		}
		log.Error("failed to get submission", sl.Err(err))
		return err
	}

	submission := details.Submission

	// If no testing report, return basic submission info
	if details.TestingReport == nil {
		return c.JSON(http.StatusOK, response.Submission{
			ID:        submission.ID,
			ProblemID: submission.ProblemID,
			Status:    submission.Status,
			Verdict:   submission.Verdict,
			Code:      submission.Code,
			Language:  submission.Language,
			CreatedAt: submission.CreatedAt,
		})
	}

	tr := details.TestingReport

	// If no failed test, return with testing report
	if details.FailedTest == nil {
		return c.JSON(http.StatusOK, response.Submission{
			ID:        submission.ID,
			ProblemID: submission.ProblemID,
			Status:    submission.Status,
			Verdict:   submission.Verdict,
			Code:      submission.Code,
			Language:  submission.Language,
			TestingReport: &response.TestingReport{
				ID:               tr.ID,
				PassedTestsCount: tr.PassedTestsCount,
				TotalTestsCount:  tr.TotalTestsCount,
				Stderr:           tr.Stderr,
				CreatedAt:        tr.CreatedAt,
			},
			CreatedAt: submission.CreatedAt,
		})
	}

	// Return with full testing report including failed test
	ftc := details.FailedTest
	return c.JSON(http.StatusOK, response.Submission{
		ID:        submission.ID,
		ProblemID: submission.ProblemID,
		Status:    submission.Status,
		Verdict:   submission.Verdict,
		Code:      submission.Code,
		Language:  submission.Language,
		TestingReport: &response.TestingReport{
			ID:               tr.ID,
			PassedTestsCount: tr.PassedTestsCount,
			TotalTestsCount:  tr.TotalTestsCount,
			FailedTest: &response.Test{
				Input:          ftc.Input,
				ExpectedOutput: ftc.Output,
				ActualOutput:   *tr.FirstFailedTestOutput,
			},
			Stderr:    tr.Stderr,
			CreatedAt: tr.CreatedAt,
		},
		CreatedAt: submission.CreatedAt,
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

	limit, ok := ExtractQueryParamInt(c, "limit")
	if !ok {
		limit = 10
	}

	offset, ok := ExtractQueryParamInt(c, "offset")
	if !ok {
		offset = 0
	}

	result, err := h.service.Submission.ListSubmissions(ctx, int32(contestID), claims.UserID, charcode, limit, offset)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCharcode):
			return Error(http.StatusBadRequest, "problem's `charcode` couldn't be longer than 2 characters")
		case errors.Is(err, service.ErrNoEntryForContest):
			return Error(http.StatusForbidden, "no entry for contest")
		default:
			log.Error("failed to list submissions", sl.Err(err))
			return err
		}
	}

	n := len(result.Submissions)
	items := make([]response.Submission, n, n)
	for i, submission := range result.Submissions {
		items[i] = response.Submission{
			ID:        submission.ID,
			ProblemID: submission.ProblemID,
			Status:    submission.Status,
			Verdict:   submission.Verdict,
			CreatedAt: submission.CreatedAt,
		}
	}

	return c.JSON(http.StatusOK, response.Pagination[response.Submission]{
		Meta: response.Meta{
			Total:   result.Total,
			Limit:   limit,
			Offset:  offset,
			HasNext: offset+limit < result.Total,
			HasPrev: offset > 0,
		},
		Items: items,
	})
}
