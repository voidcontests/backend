package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/dto/response"
	"github.com/voidcontests/backend/internal/app/service"
)

func (h *Handler) CreateEntry(c echo.Context) error {
	ctx := c.Request().Context()
	claims, _ := ExtractClaims(c)

	contestID, ok := ExtractParamInt(c, "cid")
	if !ok {
		return Error(http.StatusBadRequest, "contest ID should be an integer")
	}

	id, err := h.entryService.CreateEntry(ctx, claims.UserID, int32(contestID))
	if err != nil {
		switch err {
		case service.ErrContestNotFound:
			return Error(http.StatusNotFound, "contest not found")
		case service.ErrEntriesLimitReached:
			return Error(http.StatusForbidden, "max slots limit reached")
		case service.ErrApplicationTimeIsOver:
			return Error(http.StatusForbidden, "application time is over")
		case service.ErrEntryAlreadyExists:
			return Error(http.StatusConflict, "user already has entry for this contest")
		default:
			return err
		}
	}

	return c.JSON(http.StatusCreated, response.ID{
		ID: id,
	})
}
