package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	jwtgo "github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/request"
	"github.com/voidcontests/backend/internal/app/handler/response"
	"github.com/voidcontests/backend/internal/jwt"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	repoerr "github.com/voidcontests/backend/internal/repository/errors"
	"github.com/voidcontests/backend/pkg/requestid"
)

func (h *Handler) CreateContest(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.CreateContest"), slog.String("request_id", requestid.Get(c)))

	user := c.Get("account").(*jwtgo.Token)
	claims := user.Claims.(*jwt.CustomClaims)

	var body request.Contest
	if err := c.Bind(&body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		return c.String(http.StatusBadRequest, "bad request")
	}

	// TODO: start transaction here
	contest, err := h.repo.Contest.Create(c.Request().Context(), body.Title, body.Description, claims.Address, body.StartingAt, body.DurationMins, body.IsDraft)
	if err != nil {
		log.Error("can't create contest", sl.Err(err))
		return c.String(http.StatusInternalServerError, "internal server error")
	}

	// TODO: insert up to 10? problems in one query
	for _, p := range body.Problems {
		_, err := h.repo.Problem.Create(c.Request().Context(), contest.ID, p.Title, p.Statement, p.Difficulty, contest.CreatorAddress, p.Input, p.Answer)
		if err != nil {
			log.Error("can't create workout", sl.Err(err))
			return c.String(http.StatusInternalServerError, "internal server error")
		}
	}

	return c.JSON(http.StatusCreated, contest)
}

func (h *Handler) GetContestByID(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.GetContestByID"), slog.String("request_id", requestid.Get(c)))

	id := c.Param("id")
	contestID, err := strconv.Atoi(id)
	if err != nil {
		log.Debug("`id` param is not an integer", slog.String("id", id), sl.Err(err))
		return c.String(http.StatusBadRequest, "`id` should be integer")
	}

	contest, err := h.repo.Contest.GetByID(c.Request().Context(), int32(contestID))
	if errors.Is(repoerr.ErrContestNotFound, err) {
		return c.String(http.StatusNotFound, "contest not found")
	}
	if err != nil {
		return err
	}

	problems, err := h.repo.Contest.GetProblemset(c.Request().Context(), contest.ID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, response.Contest{
		ID:             contest.ID,
		Title:          contest.Title,
		Description:    contest.Description,
		Problemset:     problems,
		CreatorAddress: contest.CreatorAddress,
		StartingAt:     contest.StartingAt,
		DurationMins:   contest.DurationMins,
		IsDraft:        contest.IsDraft, // TODO: return `is_draft` only if contest created by request initiator
		CreatedAt:      contest.CreatedAt,
	})
}

func (h *Handler) GetContests(c echo.Context) error {
	// TODO: do not return all contests:
	// - return only active contests
	// - return by chunks

	log := slog.With(slog.String("op", "handler.GetContests"), slog.String("request_id", requestid.Get(c)))

	contests, err := h.repo.Contest.GetAll(c.Request().Context())
	if err != nil {
		log.Error("can't get contests", sl.Err(err))
		return c.String(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, response.Contests{
		Amount:   len(contests),
		Contests: contests,
	})
}
