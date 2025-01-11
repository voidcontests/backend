package handler

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/response"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/pkg/requestid"
)

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
