package handler

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/responsebody"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/pkg/requestid"
)

func (h *Handler) GetProblems(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.GetProblems"), slog.String("request_id", requestid.Get(c)))

	problems, err := h.repo.Problem.GetAll(c.Request().Context())
	if err != nil {
		log.Error("can't get contests", sl.Err(err))
		return c.String(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusCreated, responsebody.Problems{
		Amount:   len(problems),
		Problems: problems,
	})
}
