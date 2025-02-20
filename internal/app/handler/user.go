package handler

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/dto/response"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/pkg/requestid"
)

func (h *Handler) GetMe(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.GetMe"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	user, err := h.repo.User.GetByAddress(ctx, claims.Address)
	if err != nil {
		log.Debug("can't get user", sl.Err(err))
		return err
	}

	role, err := h.repo.User.GetRole(ctx, claims.ID)
	if err != nil {
		log.Debug("can't get user's role", sl.Err(err))
		return err
	}

	return c.JSON(http.StatusOK, response.Account{
		ID:      user.ID,
		Address: user.Address,
		Role: response.Role{
			Name:                 role.Name,
			CreatedProblemsLimit: role.CreatedProblemsLimit,
			CreatedContestsLimit: role.CreatedContestsLimit,
		},
	})
}
