package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/internal/repository/repoerr"
	"github.com/voidcontests/backend/pkg/requestid"
)

func (h *Handler) CreateEntry(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.CreateEntry"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	cid := c.Param("cid")
	contestID, err := strconv.Atoi(cid)
	if err != nil {
		log.Debug("`cid` param is not an integer", slog.String("cid", cid), sl.Err(err))
		return Error(http.StatusBadRequest, "`cid` should be integer")
	}

	contest, err := h.repo.Contest.GetByID(ctx, int32(contestID))
	if errors.Is(err, repoerr.ErrContestNotFound) {
		return Error(http.StatusNotFound, "contest not found")
	}
	if err != nil {
		log.Error("can't get contest by id", sl.Err(err))
		return err
	}

	entries, err := h.repo.Contest.GetEntriesCount(ctx, int32(contestID))
	if err != nil {
		log.Error("can't get entries count", sl.Err(err))
		return err
	}

	if contest.MaxEntries != 0 && entries >= contest.MaxEntries {
		log.Debug("no available slots")
		return Error(http.StatusConflict, "no available slots to join competition")
	}

	// NOTE: disallow join if: contest already finished or (already started and no late joins)
	if contest.EndTime.Before(time.Now()) || (contest.StartTime.Before(time.Now()) && !contest.AllowLateJoin) {
		return Error(http.StatusForbidden, "application time is over")
	}

	entry, err := h.repo.Entry.Get(ctx, int32(contestID), claims.ID)
	if err != nil && !errors.Is(err, repoerr.ErrEntryNotFound) {
		log.Error("can't get entry", sl.Err(err))
		return err
	}
	if entry != nil {
		return Error(http.StatusConflict, "user already has entry for this contest")
	}

	_, err = h.repo.Entry.Create(ctx, int32(contestID), claims.ID)
	if err != nil {
		log.Error("can't create entry for contest", sl.Err(err))
		return err
	}

	return c.NoContent(http.StatusCreated)
}
