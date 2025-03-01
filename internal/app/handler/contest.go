package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/dto/request"
	"github.com/voidcontests/backend/internal/app/handler/dto/response"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/internal/repository/models"
	"github.com/voidcontests/backend/internal/repository/postgres/submission"
	"github.com/voidcontests/backend/internal/repository/repoerr"
	"github.com/voidcontests/backend/pkg/requestid"
	"github.com/voidcontests/backend/pkg/validate"
)

func (h *Handler) CreateContest(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.CreateContest"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	var body request.CreateContestRequest
	if err := validate.Bind(c, &body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		return Error(http.StatusBadRequest, "invalid body: missing required fields")
	}

	userrole, err := h.repo.User.GetRole(ctx, claims.ID)
	if err != nil {
		log.Error("can't get user's role", sl.Err(err))
		return err
	}

	if userrole.Name == models.RoleBanned {
		log.Debug("banned mf tried to create new contest")
		return Error(http.StatusForbidden, "you are banned from creating contests")
	}

	if userrole.Name == models.RoleLimited {
		cscount, err := h.repo.User.GetCreatedContestsCount(ctx, claims.ID)
		if err != nil {
			log.Debug("can't get created contests count", sl.Err(err))
			return err
		}

		if cscount >= int(userrole.CreatedContestsLimit) {
			return Error(http.StatusForbidden, "contests limit exceeded")
		}
	}

	occupied, err := h.repo.Contest.IsTitleOccupied(ctx, strings.ToLower(body.Title))
	if err != nil {
		log.Error("can't verify that title isn't occupied")
		return err
	}
	if occupied {
		return Error(http.StatusConflict, "title alredy taken")
	}

	// TODO: move this limitation somwhere as MAX_PROBLEMS
	if len(body.ProblemsIDs) > 6 {
		return Error(http.StatusBadRequest, "maximum about of problems in the contest is 6")
	}

	contestID, err := h.repo.Contest.CreateWithProblemIDs(ctx, claims.ID, body.Title, body.Description, body.StartTime, body.EndTime, body.DurationMins, body.MaxEntries, body.AllowLateJoin, body.KeepAsTraining, false, body.ProblemsIDs...)
	if err != nil {
		log.Error("can't create contest", sl.Err(err))
		return err
	}

	return c.JSON(http.StatusCreated, response.ContestID{
		ID: contestID,
	})
}

func (h *Handler) GetContestByID(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.GetContestByID"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	claims, authenticated := ExtractClaims(c)

	cid := c.Param("cid")
	contestID, err := strconv.Atoi(cid)
	if err != nil {
		log.Debug("`cid` param is not an integer", slog.String("cid", cid), sl.Err(err))
		return Error(http.StatusBadRequest, "`cid` should be integer")
	}

	contest, err := h.repo.Contest.GetByID(ctx, int32(contestID))
	if errors.Is(repoerr.ErrContestNotFound, err) {
		return Error(http.StatusNotFound, "contest not found")
	}
	if err != nil {
		log.Error("can't get contest by id", sl.Err(err))
		return err
	}

	if contest.IsDraft && (!authenticated || claims.ID != contest.CreatorID) {
		return Error(http.StatusNotFound, "contest not found")
	}

	if contest.EndTime.Before(time.Now()) && !contest.KeepAsTraining {
		return Error(http.StatusNotFound, "contest not found")
	}

	problems, err := h.repo.Contest.GetProblemset(ctx, contest.ID)
	if err != nil {
		log.Error("can't get contest problemset", sl.Err(err))
		return err
	}

	n := len(problems)
	cdetailed := response.ContestDetailed{
		ID:          contest.ID,
		Title:       contest.Title,
		Description: contest.Description,
		Problems:    make([]response.ProblemListItem, n, n),
		Creator: response.User{
			ID:      contest.CreatorID,
			Address: contest.CreatorAddress,
		},
		Participants:  contest.Participants,
		StartTime:     contest.StartTime,
		EndTime:       contest.EndTime,
		DurationMins:  contest.DurationMins,
		MaxEntries:    contest.MaxEntries,
		AllowLateJoin: contest.AllowLateJoin,
		IsDraft:       contest.IsDraft,
		CreatedAt:     contest.CreatedAt,
	}

	for i := range n {
		cdetailed.Problems[i] = response.ProblemListItem{
			ID:         problems[i].ID,
			Charcode:   problems[i].Charcode,
			Title:      problems[i].Title,
			Difficulty: problems[i].Difficulty,
			Writer: response.User{
				ID:      problems[i].WriterID,
				Address: problems[i].WriterAddress,
			},
		}
	}

	// NOTE: Return contest without problem submissions
	// statuses if user is not authenticated
	if !authenticated {
		return c.JSON(http.StatusOK, cdetailed)
	}

	entry, err := h.repo.Entry.Get(ctx, contest.ID, claims.ID)
	if err != nil && !errors.Is(err, repoerr.ErrEntryNotFound) {
		log.Error("can't get entry", sl.Err(err))
		return err
	}
	if errors.Is(err, repoerr.ErrEntryNotFound) {
		return c.JSON(http.StatusOK, cdetailed)
	}

	cdetailed.IsParticipant = true

	submissions, err := h.repo.Submission.GetForEntry(ctx, entry.ID)
	if err != nil {
		log.Error("can't get submissions", sl.Err(err))
		return err
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
		case submission.VerdictWrongAnswer:
			cdetailed.Problems[i].Status = "tried"
		}
	}

	return c.JSON(http.StatusOK, cdetailed)
}

func (h *Handler) GetCreatedContests(c echo.Context) error {
	// TODO: do not return all contests:
	// - return only active contests
	// - return by chunks (pages)

	log := slog.With(slog.String("op", "handler.GetCreatedContests"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	contests, err := h.repo.Contest.GetWithCreatorID(ctx, claims.ID)
	if err != nil {
		log.Error("can't get created contests", sl.Err(err))
		return err
	}

	filtered := make([]response.ContestListItem, 0)
	for _, c := range contests {
		item := response.ContestListItem{
			ID: c.ID,
			Creator: response.User{
				ID:      c.CreatorID,
				Address: c.CreatorAddress,
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

	log := slog.With(slog.String("op", "handler.GetContests"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	contests, err := h.repo.Contest.GetAll(ctx)
	if err != nil {
		log.Error("can't get contests", sl.Err(err))
		return err
	}

	filtered := make([]response.ContestListItem, 0)
	for _, c := range contests {
		if c.IsDraft {
			continue
		}
		if c.EndTime.Before(time.Now()) && !c.KeepAsTraining {
			continue
		}

		item := response.ContestListItem{
			ID: c.ID,
			Creator: response.User{
				ID:      c.CreatorID,
				Address: c.CreatorAddress,
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
	log := slog.With(slog.String("op", "handler.GetLeaderboard"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	cid := c.Param("cid")
	contestID, err := strconv.Atoi(cid)
	if err != nil {
		log.Debug("`cid` param is not an integer", slog.String("cid", cid), sl.Err(err))
		return Error(http.StatusBadRequest, "`cid` should be integer")
	}

	// TODO: get all users that are participating in current contest
	// and return their points (if no submisisons - 0)
	leaderboard, err := h.repo.Contest.GetLeaderboard(ctx, contestID)
	if err != nil {
		log.Error("can't get leaderboard", sl.Err(err))
		return err
	}

	sort.Slice(leaderboard, func(i, j int) bool {
		// NOTE: points in non-ascending order
		return leaderboard[i].Points > leaderboard[j].Points
	})

	return c.JSON(http.StatusOK, map[string]any{
		"data": leaderboard,
	})
}
