package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/dto/request"
	"github.com/voidcontests/backend/internal/app/handler/dto/response"
	"github.com/voidcontests/backend/internal/repository/models"
	"github.com/voidcontests/backend/internal/repository/postgres/submission"
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

	// TODO: move this limitation somwhere as MAX_PROBLEMS
	if len(body.ProblemsIDs) > 6 {
		return Error(http.StatusBadRequest, "maximum about of problems in the contest is 6")
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

	cid := c.Param("cid")
	contestID, err := strconv.Atoi(cid)
	if err != nil {
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

	submissions, err := h.repo.Submission.GetForEntry(ctx, entry.ID)
	if err != nil {
		return fmt.Errorf("%s: can't get submissions: %v", op, err)
	}

	verdicts := make(map[int32]string) // map problem_id -> verdict
	for _, s := range submissions {
		if v, ok := verdicts[s.ProblemID]; ok && v == submission.VerdictOK {
			continue
		}
		verdicts[s.ProblemID] = s.Verdict
	}

	for i := range n {
		switch verdicts[problems[i].ID] {
		case submission.VerdictOK:
			cdetailed.Problems[i].Status = "accepted"
		case submission.VerdictWrongAnswer, submission.VerdictRuntimeError, submission.VerdictCompilationError, submission.VerdictTimeLimitExceeded:
			cdetailed.Problems[i].Status = "tried"
		}
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

	contests, err := h.repo.Contest.GetWithCreatorID(ctx, claims.UserID)
	if err != nil {
		return fmt.Errorf("%s: can't get created contests: %v", op, err)
	}

	filtered := make([]response.ContestListItem, 0)
	for _, c := range contests {
		item := response.ContestListItem{
			ID: c.ID,
			Creator: response.User{
				ID:       c.CreatorID,
				Username: c.CreatorUsername,
			},
			Title:        c.Title,
			StartTime:    c.StartTime,
			EndTime:      c.EndTime,
			DurationMins: c.DurationMins,
			MaxEntries:   c.MaxEntries,
			Participants: c.Participants,
			CreatedAt:    c.CreatedAt,
		}
		filtered = append(filtered, item)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": filtered,
	})
}

func (h *Handler) GetContests(c echo.Context) error {
	// TODO: do not return all contests:
	// - return only active contests
	// - return by chunks (pages)

	op := "handler.GetContests"
	ctx := c.Request().Context()

	contests, err := h.repo.Contest.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("%s: can't get contests: %v", op, err)
	}

	filtered := make([]response.ContestListItem, 0)
	for _, c := range contests {
		if c.EndTime.Before(time.Now()) {
			continue
		}

		item := response.ContestListItem{
			ID: c.ID,
			Creator: response.User{
				ID:       c.CreatorID,
				Username: c.CreatorUsername,
			},
			Title:        c.Title,
			StartTime:    c.StartTime,
			EndTime:      c.EndTime,
			DurationMins: c.DurationMins,
			MaxEntries:   c.MaxEntries,
			Participants: c.Participants,
			CreatedAt:    c.CreatedAt,
		}
		filtered = append(filtered, item)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": filtered,
	})
}

func (h *Handler) GetLeaderboard(c echo.Context) error {
	op := "handler.GetLeaderboard"
	ctx := c.Request().Context()

	cid := c.Param("cid")
	contestID, err := strconv.Atoi(cid)
	if err != nil {
		return Error(http.StatusBadRequest, "contest ID should be an integer")
	}

	leaderboard, err := h.repo.Contest.GetLeaderboard(ctx, contestID)
	if err != nil {
		return fmt.Errorf("%s: can't get leaderboard: %v", op, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"data": leaderboard,
	})
}
