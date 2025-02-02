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

func (h *Handler) CreateContest(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.CreateContest"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	var body request.CreateContestRequest
	if err := validate.Bind(c, &body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		return Error(http.StatusBadRequest, "invalid body: missing required fields")
	}

	occupied, err := h.repo.Contest.IsTitleOccupied(ctx, strings.ToLower(body.Title))
	if err != nil {
		log.Error("can't verify that title isn't occupied")
		return err
	}
	if occupied {
		return Error(http.StatusConflict, "title alredy taken")
	}

	contestID, err := h.repo.Contest.Create(ctx, claims.ID, body.Title, body.Description, body.StartTime, body.EndTime, body.DurationMins, false)
	if err != nil {
		log.Error("can't create contest", sl.Err(err))
		return err
	}

	// TODO: insert contest and problem in transaction
	// TODO: insert up to 10 problems in one query (???)
	for _, p := range body.Problems {
		_, err := h.repo.Problem.Create(ctx, contestID, claims.ID, p.Title, p.Statement, p.Difficulty, p.Input, p.Answer)
		if err != nil {
			log.Error("can't create workout", sl.Err(err))
			return err
		}
	}

	return c.JSON(http.StatusCreated, response.ContestID{
		ID: contestID,
	})
}

func (h *Handler) GetContestByID(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.GetContestByID"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	claims, authenticated := ExtractClaims(c)

	cid := c.Param("cid")
	contestID, err := strconv.Atoi(cid)
	if err != nil {
		log.Debug("`cid` param is not an integer", slog.String("cid", cid), sl.Err(err))
		return Error(http.StatusBadRequest, "`cid` should be integer")
	}

	contest, err := h.repo.Contest.GetByID(ctx, int32(contestID))
	if errors.Is(repoerr.ErrContestNotFound, err) {
		return Error(http.StatusNotFound, "contest not found")
	}
	if err != nil {
		log.Error("can't get contest by id", sl.Err(err))
		return err
	}

	if contest.IsDraft && (!authenticated || claims.ID != contest.CreatorID) {
		return Error(http.StatusNotFound, "contest not found")
	}

	problems, err := h.repo.Contest.GetProblemset(ctx, contest.ID)
	if err != nil {
		log.Error("can't get contest problemset", sl.Err(err))
		return err
	}

	participants, err := h.repo.Contest.GetParticipantsCount(ctx, contest.ID)
	if err != nil {
		log.Error("can't get participants count for contest", sl.Err(err))
		return err
	}

	n := len(problems)
	cdetailed := response.ContestDetailed{
		ID:          contest.ID,
		Title:       contest.Title,
		Description: contest.Description,
		Problems:    make([]response.ProblemListItem, n, n),
		Creator: response.User{
			ID:      contest.CreatorID,
			Address: contest.CreatorAddress,
		},
		Participants: participants,
		StartTime:    contest.StartTime,
		EndTime:      contest.EndTime,
		DurationMins: contest.DurationMins,
		IsDraft:      contest.IsDraft,
	}

	for i := range n {
		cdetailed.Problems[i] = response.ProblemListItem{
			ID:        problems[i].ID,
			ContestID: problems[i].ContestID,
			Writer: response.User{
				ID:      problems[i].WriterID,
				Address: problems[i].WriterAddress,
			},
			Title:      problems[i].Title,
			Difficulty: problems[i].Difficulty,
		}
	}

	// NOTE: Return contest without problem submissions
	// statuses if user is not authenticated
	if !authenticated {
		return c.JSON(http.StatusOK, cdetailed)
	}

	entry, err := h.repo.Entry.Get(ctx, contest.ID, claims.ID)
	if err != nil && !errors.Is(err, repoerr.ErrEntryNotFound) {
		log.Error("can't get entry", sl.Err(err))
		return err
	}
	if errors.Is(err, repoerr.ErrEntryNotFound) {
		return c.JSON(http.StatusOK, cdetailed)
	}

	cdetailed.IsParticipant = true

	submissions, err := h.repo.Submission.GetForEntry(ctx, entry.ID)
	if err != nil {
		log.Error("can't get submissions", sl.Err(err))
		return err
	}

	verdicts := make(map[int32]string) // map problem_id -> verdict
	for _, s := range submissions {
		if v, ok := verdicts[s.ProblemID]; ok && v == submission.VerdictOK {
			continue
		}
		verdicts[s.ProblemID] = s.Verdict
	}

	for i := range n {
		switch verdicts[problems[i].ID] {
		case submission.VerdictOK:
			cdetailed.Problems[i].Status = "accepted"
		case submission.VerdictWrongAnswer:
			cdetailed.Problems[i].Status = "tried"
		}
	}

	return c.JSON(http.StatusOK, cdetailed)
}

func (h *Handler) GetContests(c echo.Context) error {
	// TODO: do not return all contests:
	// - return only active contests
	// - return by chunks (pages)

	log := slog.With(slog.String("op", "handler.GetContests"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	contests, err := h.repo.Contest.GetAll(ctx)
	if err != nil {
		log.Error("can't get contests", sl.Err(err))
		return err
	}

	filtered := make([]response.ContestListItem, 0)
	for _, c := range contests {
		if c.IsDraft {
			continue
		}
		if c.StartTime.Add(time.Minute * time.Duration(c.DurationMins)).Before(time.Now()) {
			continue
		}

		item := response.ContestListItem{
			ID: c.ID,
			Creator: response.User{
				ID:      c.CreatorID,
				Address: c.CreatorAddress,
			},
			Title:        c.Title,
			StartTime:    c.StartTime,
			EndTime:      c.EndTime,
			DurationMins: c.DurationMins,
		}
		filtered = append(filtered, item)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": filtered,
	})
}

func (h *Handler) CreateEntry(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.CreateEntry"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	cid := c.Param("cid")
	contestID, err := strconv.Atoi(cid)
	if err != nil {
		log.Debug("`cid` param is not an integer", slog.String("cid", cid), sl.Err(err))
		return Error(http.StatusBadRequest, "`cid` should be integer")
	}

	contest, err := h.repo.Contest.GetByID(ctx, int32(contestID))
	if errors.Is(err, repoerr.ErrContestNotFound) {
		return Error(http.StatusNotFound, "contest not found")
	}
	if err != nil {
		log.Error("can't get contest by id", sl.Err(err))
		return err
	}

	if contest.StartTime.Before(time.Now()) {
		return Error(http.StatusForbidden, "application time is over")
	}

	entry, err := h.repo.Entry.Get(ctx, int32(contestID), claims.ID)
	if err != nil && !errors.Is(err, repoerr.ErrEntryNotFound) {
		log.Error("can't get entry", sl.Err(err))
		return err
	}
	if entry != nil {
		return Error(http.StatusConflict, "user already has entry for this contest")
	}

	_, err = h.repo.Entry.Create(ctx, int32(contestID), claims.ID)
	if err != nil {
		log.Error("can't create entry for contest", sl.Err(err))
		return err
	}

	return c.NoContent(http.StatusCreated)
}

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

	pid := c.Param("pid")
	problemID, err := strconv.Atoi(pid)
	if err != nil {
		log.Debug("`pid` param is not an integer", slog.String("pid", pid), sl.Err(err))
		return Error(http.StatusBadRequest, "`pid` should be integer")
	}

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

	answer, err := h.repo.Problem.GetAnswer(ctx, int32(problemID))
	if errors.Is(err, repoerr.ErrProblemNotFound) {
		return Error(http.StatusNotFound, "problem not found")
	}
	if err != nil {
		log.Error("can't get problem", sl.Err(err))
		return err
	}

	var verdict string
	if answer != body.Answer {
		verdict = submission.VerdictWrongAnswer
	} else {
		verdict = submission.VerdictOK
	}

	submission, err := h.repo.Submission.Create(ctx, entry.ID, int32(problemID), verdict, body.Answer)
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

	pid := c.Param("pid")
	problemID, err := strconv.Atoi(pid)
	if err != nil {
		log.Debug("`pid` param is not an integer", slog.String("pid", pid), sl.Err(err))
		return Error(http.StatusBadRequest, "`pid` should be integer")
	}

	entry, err := h.repo.Entry.Get(ctx, int32(contestID), claims.ID)
	if errors.Is(err, repoerr.ErrEntryNotFound) {
		return Error(http.StatusForbidden, "no entry for contest")
	}
	if err != nil {
		log.Error("can't get entry", sl.Err(err))
		return err
	}

	submissions, err := h.repo.Submission.GetForProblem(ctx, entry.ID, int32(problemID))
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
