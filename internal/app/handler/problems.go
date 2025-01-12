package handler

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/response"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/pkg/requestid"
)

func (h *Handler) GetProblems(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.GetProblems"), slog.String("request_id", requestid.Get(c)))

	// TODO: return problems splitted by chunks
	detailed_problems, err := h.repo.Problem.GetAll(c.Request().Context())
	if err != nil {
		log.Error("can't get contests", sl.Err(err))
		return err
	}

	problems := make([]response.Problem, len(detailed_problems), len(detailed_problems))
	for i, p := range problems {
		problems[i] = response.Problem{
			ID:            p.ID,
			ContestID:     p.ContestID,
			Title:         p.Title,
			Difficulty:    p.Difficulty,
			WriterAddress: p.WriterAddress,
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": problems,
	})
}
