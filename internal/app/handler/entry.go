package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/api/internal/app/handler/dto/response"
	"github.com/voidcontests/api/internal/app/service"
)

func (h *Handler) CreateEntry(c echo.Context) error {
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	contestID, ok := ExtractParamInt(c, "cid")
	if !ok {
		return Error(http.StatusBadRequest, "contest ID should be an integer")
	}

	err := h.service.Entry.CreateEntry(ctx, contestID, claims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrContestNotFound):
			return Error(http.StatusNotFound, "contest not found")
		case errors.Is(err, service.ErrMaxSlotsReached):
			return Error(http.StatusConflict, "max slots limit reached")
		case errors.Is(err, service.ErrApplicationTimeOver):
			return Error(http.StatusForbidden, "application time is over")
		case errors.Is(err, service.ErrEntryAlreadyExists):
			return Error(http.StatusConflict, "user already has entry for this contest")
		default:
			return err
		}
	}

	return c.NoContent(http.StatusCreated)
}

func (h *Handler) GetEntry(c echo.Context) error {
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	contestID, ok := ExtractParamInt(c, "cid")
	if !ok {
		return Error(http.StatusBadRequest, "contest ID should be an integer")
	}

	entry, err := h.service.Entry.GetEntry(ctx, contestID, claims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEntryNotFound):
			return Error(http.StatusNotFound, "entry not found")
		case errors.Is(err, service.ErrContestNotFound):
			return Error(http.StatusNotFound, "contest not found")
		default:
			return err
		}
	}

	return c.JSON(http.StatusOK, response.Entry{
		ID:        entry.ID,
		ContestID: entry.ContestID,
		UserID:    entry.UserID,
		IsPaid:    entry.IsPaid,
		CreatedAt: entry.CreatedAt,
	})
}
