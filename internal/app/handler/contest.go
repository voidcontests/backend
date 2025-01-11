package handler

import (
	"log/slog"
	"net/http"

	jwtgo "github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/requestbody"
	"github.com/voidcontests/backend/internal/jwt"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/pkg/requestid"
)

func (h *Handler) CreateContest(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.CreateContest"), slog.String("request_id", requestid.Get(c)))

	user := c.Get("account").(*jwtgo.Token)
	claims := user.Claims.(*jwt.CustomClaims)

	var body requestbody.Contest
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
