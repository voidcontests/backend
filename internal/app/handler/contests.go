package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/requestbody"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/pkg/requestid"
)

func (h *Handler) CreateContest(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.CreateContest"), slog.String("request_id", requestid.Get(c)))

	var body requestbody.Contest
	if err := c.Bind(&body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		return c.String(http.StatusBadRequest, "bad request")
	}

	address := "UQBhHFb_9Df--VGocjB2qUBg3UEuEUvF_wgtFEd_k7xRDE0U" // TODO: extract fron auth header
	startingAt := time.Now().Add(24 * time.Hour)                  // TODO: extract from

	// TODO: start transaction here
	contest, err := h.repo.Contest.Create(c.Request().Context(), body.Title, body.Description, address, startingAt, body.DurationMins, body.IsDraft)
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
