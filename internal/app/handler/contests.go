package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	jwtgo "github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/dto/request"
	"github.com/voidcontests/backend/internal/app/handler/dto/response"
	"github.com/voidcontests/backend/internal/jwt"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/internal/repository/postgres/submission"
	"github.com/voidcontests/backend/internal/repository/repoerr"
	"github.com/voidcontests/backend/pkg/requestid"
	"github.com/voidcontests/backend/pkg/validate"
)

func (h *Handler) CreateContest(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.CreateContest"), slog.String("request_id", requestid.Get(c)))

	user := c.Get("account").(*jwtgo.Token)
	claims := user.Claims.(*jwt.CustomClaims)

	var body request.CreateContestRequest
	if err := validate.Bind(c, &body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		return Error(http.StatusBadRequest, "invalid body: missing required fields")
	}
	if !body.IsDraft && (body.StartingAt.IsZero() || body.DurationMins == 0 || len(body.Problems) == 0) {
		return Error(http.StatusBadRequest, "invalid body: only draft contest may have empty fields")
	}

	log.Info("msg string", slog.Any("key string", body))

	// TODO: start transaction here
	contest, err := h.repo.Contest.Create(c.Request().Context(), claims.ID, body.Title, body.Description, body.StartingAt, body.DurationMins, body.IsDraft)
	if err != nil {
		log.Error("can't create contest", sl.Err(err))
		return err
	}

	// TODO: insert up to 10? problems in one query
	for _, p := range body.Problems {
		_, err := h.repo.Problem.Create(c.Request().Context(), contest.ID, contest.CreatorID, p.Title, p.Statement, p.Difficulty, p.Input, p.Answer)
		if err != nil {
			log.Error("can't create workout", sl.Err(err))
			return err
		}
	}

	return c.NoContent(http.StatusCreated)
}

func (h *Handler) GetContestByID(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.GetContestByID"), slog.String("request_id", requestid.Get(c)))

	data := c.Get("account")
	var claims *jwt.CustomClaims
	if data != nil {
		claims = data.(*jwt.CustomClaims)
	}

	cid := c.Param("cid")
	contestID, err := strconv.Atoi(cid)
	if err != nil {
		log.Debug("`cid` param is not an integer", slog.String("cid", cid), sl.Err(err))
		return Error(http.StatusBadRequest, "`cid` should be integer")
	}

	contest, err := h.repo.Contest.GetByID(c.Request().Context(), int32(contestID))
	if errors.Is(repoerr.ErrContestNotFound, err) {
		return Error(http.StatusNotFound, "contest not found")
	}
	if err != nil {
		log.Error("can't get contest by id", sl.Err(err))
		return err
	}

	problems, err := h.repo.Contest.GetProblemset(c.Request().Context(), contest.ID)
	if err != nil {
		log.Error("can't get contest problemset", sl.Err(err))
		return err
	}

	n := len(problems)
	problemset := make([]response.ProblemListItem, n, n)
	for i := range n {
		problemset[i] = response.ProblemListItem{
			ID:         problems[i].ID,
			ContestID:  problems[i].ContestID,
			WriterID:   problems[i].WriterID,
			Title:      problems[i].Title,
			Difficulty: problems[i].Difficulty,
		}
	}

	// TODO: Add problem's status: solved, tried or none
	if contest.IsDraft && (claims == nil || claims.ID != contest.CreatorID) {
		return Error(http.StatusNotFound, "contest not found")
	}

	return c.JSON(http.StatusOK, response.ContestDetailed{
		ID:           contest.ID,
		Title:        contest.Title,
		Description:  contest.Description,
		Problems:     problemset,
		CreatorID:    contest.CreatorID,
		StartingAt:   contest.StartingAt,
		DurationMins: contest.DurationMins,
		IsDraft:      contest.IsDraft,
	})
}

func (h *Handler) GetContests(c echo.Context) error {
	// TODO: do not return all contests:
	// - return only active contests
	// - return by chunks (pages)

	log := slog.With(slog.String("op", "handler.GetContests"), slog.String("request_id", requestid.Get(c)))

	contests, err := h.repo.Contest.GetAll(c.Request().Context())
	if err != nil {
		log.Error("can't get contests", sl.Err(err))
		return err
	}

	n := len(contests)
	published := make([]response.ContestListItem, n, n)
	for i, c := range contests {
		if !c.IsDraft {
			published[i] = response.ContestListItem{
				ID:           c.ID,
				CreatorID:    c.CreatorID,
				Title:        c.Title,
				StartingAt:   c.StartingAt,
				DurationMins: c.DurationMins,
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": published,
	})
}

func (h *Handler) CreateEntry(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.CreateEntry"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	user := c.Get("account").(*jwtgo.Token)
	claims := user.Claims.(*jwt.CustomClaims)

	cid := c.Param("cid")
	contestID, err := strconv.Atoi(cid)
	if err != nil {
		log.Debug("`cid` param is not an integer", slog.String("cid", cid), sl.Err(err))
		return Error(http.StatusBadRequest, "`cid` should be integer")
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

	user := c.Get("account").(*jwtgo.Token)
	claims := user.Claims.(*jwt.CustomClaims)

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

	log.Info("msg string", slog.Any("key string", body))

	entry, err := h.repo.Entry.Get(ctx, int32(contestID), claims.ID)
	if err != nil {
		log.Error("can't get entry", sl.Err(err))
		return err
	}

	answer, err := h.repo.Problem.GetAnswer(ctx, int32(problemID))
	if err != nil {
		log.Error("can't get problem", sl.Err(err))
		return err
	}

	var verdict submission.Verdict
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

	user := c.Get("account").(*jwtgo.Token)
	claims := user.Claims.(*jwt.CustomClaims)

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
	if err != nil {
		log.Error("can't get entry", sl.Err(err))
		return err
	}

	submissions, err := h.repo.Submission.GetForProblem(ctx, claims.ID, entry.ID, int32(problemID))
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
