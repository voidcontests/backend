package handler

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/dto/response"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/pkg/requestid"
)

func (h *Handler) GetProblems(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.GetProblems"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	// TODO: return problems splitted by chunks
	ps, err := h.repo.Problem.GetAll(ctx)
	if err != nil {
		log.Error("can't get contests", sl.Err(err))
		return err
	}

	n := len(ps)
	problems := make([]response.ProblemListItem, n, n)
	for i, p := range ps {
		problems[i] = response.ProblemListItem{
			ID:         p.ID,
			ContestID:  p.ContestID,
			WriterID:   p.WriterID,
			Title:      p.Title,
			Difficulty: p.Difficulty,
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": problems,
	})
}
