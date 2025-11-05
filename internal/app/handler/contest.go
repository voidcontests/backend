package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/api/internal/app/handler/dto/request"
	"github.com/voidcontests/api/internal/app/handler/dto/response"
	"github.com/voidcontests/api/internal/app/service"
	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/pkg/validate"
)

func (h *Handler) CreateContest(c echo.Context) error {
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	var body request.CreateContest
	if err := validate.Bind(c, &body); err != nil {
		return Error(http.StatusBadRequest, "invalid body: missing required fields")
	}

	id, err := h.service.Contest.CreateContest(ctx, service.CreateContestParams{
		UserID:             claims.UserID,
		Title:              body.Title,
		Description:        body.Description,
		AwardType:          body.AwardType,
		EntryPriceTonNanos: body.EntryPriceTonNanos,
		StartTime:          body.StartTime,
		EndTime:            body.EndTime,
		DurationMins:       body.DurationMins,
		MaxEntries:         body.MaxEntries,
		AllowLateJoin:      body.AllowLateJoin,
		ProblemIDs:         body.ProblemsIDs,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserBanned):
			return Error(http.StatusForbidden, "you are banned from creating contests")
		case errors.Is(err, service.ErrContestsLimitExceeded):
			return Error(http.StatusForbidden, "contests limit exceeded")
		default:
			return err
		}
	}

	return c.JSON(http.StatusCreated, response.ID{
		ID: id,
	})
}

func (h *Handler) GetContestByID(c echo.Context) error {
	ctx := c.Request().Context()

	claims, authenticated := ExtractClaims(c)

	contestID, ok := ExtractParamInt(c, "cid")
	if !ok {
		return Error(http.StatusBadRequest, "contest ID should be an integer")
	}

	details, err := h.service.Contest.GetContestByID(ctx, contestID, claims.UserID, authenticated)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrContestNotFound):
			return Error(http.StatusNotFound, "contest not found")
		case errors.Is(err, service.ErrContestFinished):
			return Error(http.StatusNotFound, "contest not found")
		default:
			return err
		}
	}

	contest := details.Contest
	n := len(details.Problems)
	cdetailed := response.ContestDetailed{
		ID:                 contest.ID,
		Title:              contest.Title,
		Description:        contest.Description,
		AwardType:          contest.AwardType,
		EntryPriceTonNanos: contest.EntryPriceTonNanos,
		Creator: response.User{
			ID:       contest.CreatorID,
			Username: contest.CreatorUsername,
		},
		Problems:           make([]response.ContestProblemListItem, n, n),
		Participants:       contest.ParticipantsCount,
		StartTime:          contest.StartTime,
		EndTime:            contest.EndTime,
		DurationMins:       contest.DurationMins,
		MaxEntries:         contest.MaxEntries,
		AllowLateJoin:      contest.AllowLateJoin,
		IsParticipant:      details.IsParticipant,
		SubmissionDeadline: details.SubmissionDeadline,
		Prizes: response.Prizes{
			Nanos: details.PrizeNanosTON,
		},
		CreatedAt: contest.CreatedAt,
	}

	for i := range n {
		p := details.Problems[i]
		cdetailed.Problems[i] = response.ContestProblemListItem{
			ID:       p.ID,
			Charcode: p.Charcode,
			Writer: response.User{
				ID:       p.WriterID,
				Username: p.WriterUsername,
			},
			Title:         p.Title,
			Difficulty:    p.Difficulty,
			TimeLimitMS:   p.TimeLimitMS,
			MemoryLimitMB: p.MemoryLimitMB,
			Checker:       p.Checker,
			CreatedAt:     p.CreatedAt,
			Status:        details.ProblemStatuses[p.ID],
		}
	}

	return c.JSON(http.StatusOK, cdetailed)
}

func (h *Handler) GetCreatedContests(c echo.Context) error {
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	limit, ok := ExtractQueryParamInt(c, "limit")
	if !ok {
		limit = 10
	}

	offset, ok := ExtractQueryParamInt(c, "offset")
	if !ok {
		offset = 0
	}

	result, err := h.service.Contest.ListCreatedContests(ctx, claims.UserID, limit, offset)
	if err != nil {
		return err
	}

	items := make([]response.ContestListItem, 0)
	for _, contest := range result.Contests {
		item := response.ContestListItem{
			ID: contest.ID,
			Creator: response.User{
				ID:       contest.CreatorID,
				Username: contest.CreatorUsername,
			},
			Title:              contest.Title,
			AwardType:          contest.AwardType,
			EntryPriceTonNanos: contest.EntryPriceTonNanos,
			StartTime:          contest.StartTime,
			EndTime:            contest.EndTime,
			DurationMins:       contest.DurationMins,
			MaxEntries:         contest.MaxEntries,
			Participants:       contest.ParticipantsCount,
			CreatedAt:          contest.CreatedAt,
		}
		items = append(items, item)
	}

	return c.JSON(http.StatusOK, response.Pagination[response.ContestListItem]{
		Meta: response.Meta{
			Total:   result.Total,
			Limit:   limit,
			Offset:  offset,
			HasNext: offset+limit < result.Total,
			HasPrev: offset > 0,
		},
		Items: items,
	})
}

func (h *Handler) GetContests(c echo.Context) error {
	ctx := c.Request().Context()

	limit, ok := ExtractQueryParamInt(c, "limit")
	if !ok {
		limit = 10
	}

	offset, ok := ExtractQueryParamInt(c, "offset")
	if !ok || offset < 0 {
		offset = 0
	}

	filters := models.ContestFilters{}

	if creatorID, ok := ExtractQueryParamInt(c, "creator_id"); ok {
		if creatorID > 0 {
			return Error(http.StatusBadRequest, "creator_id should be a valid integer, greater 0")
		}
		filters.CreatorID = creatorID
	}

	if title := c.QueryParam("title"); title != "" {
		filters.Title = title
	}

	result, err := h.service.Contest.ListAllContests(ctx, limit, offset, filters)
	if err != nil {
		return err
	}

	items := make([]response.ContestListItem, 0)
	for _, contest := range result.Contests {
		item := response.ContestListItem{
			ID: contest.ID,
			Creator: response.User{
				ID:       contest.CreatorID,
				Username: contest.CreatorUsername,
			},
			Title:              contest.Title,
			AwardType:          contest.AwardType,
			EntryPriceTonNanos: contest.EntryPriceTonNanos,
			StartTime:          contest.StartTime,
			EndTime:            contest.EndTime,
			DurationMins:       contest.DurationMins,
			MaxEntries:         contest.MaxEntries,
			Participants:       contest.ParticipantsCount,
			CreatedAt:          contest.CreatedAt,
		}
		items = append(items, item)
	}

	return c.JSON(http.StatusOK, response.Pagination[response.ContestListItem]{
		Meta: response.Meta{
			Total:   result.Total,
			Limit:   limit,
			Offset:  offset,
			HasNext: offset+limit < result.Total,
			HasPrev: offset > 0,
		},
		Items: items,
	})
}

func (h *Handler) GetLeaderboard(c echo.Context) error {
	ctx := c.Request().Context()

	contestID, ok := ExtractParamInt(c, "cid")
	if !ok {
		return Error(http.StatusBadRequest, "contest ID should be an integer")
	}

	limit, ok := ExtractQueryParamInt(c, "limit")
	if !ok {
		limit = 50
	}

	offset, ok := ExtractQueryParamInt(c, "offset")
	if !ok {
		offset = 0
	}

	result, err := h.service.Contest.GetLeaderboard(ctx, contestID, limit, offset)
	if err != nil {
		if errors.Is(err, service.ErrContestNotFound) {
			return Error(http.StatusNotFound, "contest not found")
		}
		return err
	}

	return c.JSON(http.StatusOK, response.Pagination[models.LeaderboardEntry]{
		Meta: response.Meta{
			Total:   result.Total,
			Limit:   limit,
			Offset:  offset,
			HasNext: offset+limit < result.Total,
			HasPrev: offset > 0,
		},
		Items: result.Leaderboard,
	})
}
