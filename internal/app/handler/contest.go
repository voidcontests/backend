package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/dto/request"
	"github.com/voidcontests/backend/internal/app/handler/dto/response"
	"github.com/voidcontests/backend/internal/repository/models"
	"github.com/voidcontests/backend/pkg/validate"
)

func (h *Handler) CreateContest(c echo.Context) error {
	op := "handler.CreateContest"
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	var body request.CreateContestRequest
	if err := validate.Bind(c, &body); err != nil {
		return Error(http.StatusBadRequest, "invalid body: missing required fields")
	}

	userrole, err := h.repo.User.GetRole(ctx, claims.UserID)
	if err != nil {
		return fmt.Errorf("%s: can't get role: %v", op, err)
	}

	if userrole.Name == models.RoleBanned {
		return Error(http.StatusForbidden, "you are banned from creating contests")
	}

	if userrole.Name == models.RoleLimited {
		cscount, err := h.repo.User.GetCreatedContestsCount(ctx, claims.UserID)
		if err != nil {
			return fmt.Errorf("%s: can't get created contests count: %v", op, err)
		}

		if cscount >= int(userrole.CreatedContestsLimit) {
			return Error(http.StatusForbidden, "contests limit exceeded")
		}
	}

	occupied, err := h.repo.Contest.IsTitleOccupied(ctx, strings.ToLower(body.Title))
	if err != nil {
		return fmt.Errorf("%s: can't verify that title isn't occupied: %v", op, err)
	}
	if occupied {
		return Error(http.StatusConflict, "title alredy taken")
	}

	contestID, err := h.repo.Contest.CreateWithProblemIDs(ctx, claims.UserID, body.Title, body.Description, body.StartTime, body.EndTime, body.DurationMins, body.MaxEntries, body.AllowLateJoin, body.ProblemsIDs)
	if err != nil {
		return fmt.Errorf("%s: can't create contest: %v", op, err)
	}

	return c.JSON(http.StatusCreated, response.ID{
		ID: contestID,
	})
}

func (h *Handler) GetContestByID(c echo.Context) error {
	op := "handler.GetContestByID"
	ctx := c.Request().Context()

	claims, authenticated := ExtractClaims(c)

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

	// TODO: allow check previuos contests
	if contest.EndTime.Before(time.Now()) {
		return Error(http.StatusNotFound, "contest not found")
	}

	problems, err := h.repo.Contest.GetProblemset(ctx, contest.ID)
	if err != nil {
		return fmt.Errorf("%s: can't get problemset: %v", op, err)
	}

	n := len(problems)
	cdetailed := response.ContestDetailed{
		ID:          contest.ID,
		Title:       contest.Title,
		Description: contest.Description,
		Problems:    make([]response.ProblemListItem, n, n),
		Creator: response.User{
			ID:       contest.CreatorID,
			Username: contest.CreatorUsername,
		},
		Participants:  contest.Participants,
		StartTime:     contest.StartTime,
		EndTime:       contest.EndTime,
		DurationMins:  contest.DurationMins,
		MaxEntries:    contest.MaxEntries,
		AllowLateJoin: contest.AllowLateJoin,
		CreatedAt:     contest.CreatedAt,
	}

	for i := range n {
		cdetailed.Problems[i] = response.ProblemListItem{
			ID:         problems[i].ID,
			Charcode:   problems[i].Charcode,
			Title:      problems[i].Title,
			Difficulty: problems[i].Difficulty,
			Writer: response.User{
				ID:       problems[i].WriterID,
				Username: problems[i].WriterUsername,
			},
		}
	}

	// NOTE: Return contest without problem submissions
	// statuses if user is not authenticated
	if !authenticated {
		return c.JSON(http.StatusOK, cdetailed)
	}

	entry, err := h.repo.Entry.Get(ctx, contest.ID, claims.UserID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: can't get entry: %v", op, err)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return c.JSON(http.StatusOK, cdetailed)
	}

	cdetailed.IsParticipant = true

	statuses, err := h.repo.Submission.GetProblemStatuses(ctx, entry.ID)
	if err != nil {
		return fmt.Errorf("%s: can't get submissions: %v", op, err)
	}

	for i := range n {
		problemID := cdetailed.Problems[i].ID
		cdetailed.Problems[i].Status = statuses[problemID]
	}

	return c.JSON(http.StatusOK, cdetailed)
}

func (h *Handler) GetCreatedContests(c echo.Context) error {
	// TODO: do not return all contests:
	// - return only active contests
	// - return by chunks (pages)

	op := "handler.GetCreatedContests"
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

	contests, total, err := h.repo.Contest.GetWithCreatorID(ctx, claims.UserID, limit, offset)
	if err != nil {
		return fmt.Errorf("%s: can't get created contests: %v", op, err)
	}

	items := make([]response.ContestListItem, 0)
	for _, contest := range contests {
		item := response.ContestListItem{
			ID: contest.ID,
			Creator: response.User{
				ID:       contest.CreatorID,
				Username: contest.CreatorUsername,
			},
			Title:        contest.Title,
			StartTime:    contest.StartTime,
			EndTime:      contest.EndTime,
			DurationMins: contest.DurationMins,
			MaxEntries:   contest.MaxEntries,
			Participants: contest.Participants,
			CreatedAt:    contest.CreatedAt,
		}
		items = append(items, item)
	}

	return c.JSON(http.StatusOK, response.Pagination[response.ContestListItem]{
		Meta: response.Meta{
			Total:   total,
			Limit:   limit,
			Offset:  offset,
			HasNext: offset+limit < total,
			HasPrev: offset > 0,
		},
		Items: items,
	})
}

func (h *Handler) GetContests(c echo.Context) error {
	op := "handler.GetContests"
	ctx := c.Request().Context()

	limit, ok := ExtractQueryParamInt(c, "limit")
	if !ok {
		limit = 10
	}

	offset, ok := ExtractQueryParamInt(c, "offset")
	if !ok {
		offset = 0
	}

	contests, total, err := h.repo.Contest.ListAll(ctx, limit, offset)
	if err != nil {
		return fmt.Errorf("%s: can't get contests: %v", op, err)
	}

	items := make([]response.ContestListItem, 0)
	for _, contest := range contests {
		item := response.ContestListItem{
			ID: contest.ID,
			Creator: response.User{
				ID:       contest.CreatorID,
				Username: contest.CreatorUsername,
			},
			Title:        contest.Title,
			StartTime:    contest.StartTime,
			EndTime:      contest.EndTime,
			DurationMins: contest.DurationMins,
			MaxEntries:   contest.MaxEntries,
			Participants: contest.Participants,
			CreatedAt:    contest.CreatedAt,
		}
		items = append(items, item)
	}

	return c.JSON(http.StatusOK, response.Pagination[response.ContestListItem]{
		Meta: response.Meta{
			Total:   total,
			Limit:   limit,
			Offset:  offset,
			HasNext: offset+limit < total,
			HasPrev: offset > 0,
		},
		Items: items,
	})
}

func (h *Handler) GetLeaderboard(c echo.Context) error {
	op := "handler.GetLeaderboard"
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

	leaderboard, total, err := h.repo.Contest.GetLeaderboard(ctx, contestID, limit, offset)
	if err != nil {
		return fmt.Errorf("%s: can't get leaderboard: %v", op, err)
	}

	return c.JSON(http.StatusOK, response.Pagination[models.LeaderboardEntry]{
		Meta: response.Meta{
			Total:   total,
			Limit:   limit,
			Offset:  offset,
			HasNext: offset+limit < total,
			HasPrev: offset > 0,
		},
		Items: leaderboard,
	})
}
