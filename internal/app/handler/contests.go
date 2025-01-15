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
	repoerr "github.com/voidcontests/backend/internal/repository/errors"
	"github.com/voidcontests/backend/pkg/requestid"
)

func (h *Handler) CreateContest(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.CreateContest"), slog.String("request_id", requestid.Get(c)))

	user := c.Get("account").(*jwtgo.Token)
	claims := user.Claims.(*jwt.CustomClaims)

	var body request.CreateContestRequest
	if err := c.Bind(&body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		return Error(http.StatusBadRequest, "invalid body")
	}

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

	id := c.Param("id")
	contestID, err := strconv.Atoi(id)
	if err != nil {
		log.Debug("`id` param is not an integer", slog.String("id", id), sl.Err(err))
		return Error(http.StatusBadRequest, "`id` should be integer")
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

	return c.JSON(http.StatusOK, response.Contest{
		ID:           contest.ID,
		Title:        contest.Title,
		Description:  contest.Description,
		Problems:     problemset,
		CreatorID:    contest.CreatorID,
		StartingAt:   contest.StartingAt,
		DurationMins: contest.DurationMins,
		IsDraft:      contest.IsDraft, // TODO: return `is_draft` only if contest created by request initiator
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
