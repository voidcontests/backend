package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
)

func (h *Handler) CreateEntry(c echo.Context) error {
	op := "handler.CreateEntry"
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	contestID, ok := ExtractParamInt(c, "cid")
	if !ok {
		return Error(http.StatusBadRequest, "contest ID should be an integer")
	}

	contest, err := h.repo.Contest.GetByID(ctx, int32(contestID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Error(http.StatusNotFound, "contest not found")
	}
	if err != nil {
		return fmt.Errorf("%s: can't get contest: %v", op, err)
	}

	entries, err := h.repo.Contest.GetEntriesCount(ctx, int32(contestID))
	if err != nil {
		return fmt.Errorf("%s: can't get entries: %v", op, err)
	}

	if contest.MaxEntries != 0 && entries >= contest.MaxEntries {
		return Error(http.StatusConflict, "max slots limit reached")
	}

	// NOTE: disallow join if: contest already finished or (already started and no late joins)
	if contest.EndTime.Before(time.Now()) || (contest.StartTime.Before(time.Now()) && !contest.AllowLateJoin) {
		return Error(http.StatusForbidden, "application time is over")
	}

	_, err = h.repo.Entry.Get(ctx, int32(contestID), claims.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = h.repo.Entry.Create(ctx, int32(contestID), claims.UserID)
		if err != nil {
			return fmt.Errorf("%s: can't create entry: %v", op, err)
		}

		return c.NoContent(http.StatusCreated)
	}
	if err != nil {
		return fmt.Errorf("%s: can't get entry: %v", op, err)
	}

	return Error(http.StatusConflict, "user already has entry for this contest")
}
