package contest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/voidcontests/backend/internal/repository/models"
	"github.com/voidcontests/backend/internal/repository/repoerr"
)

type Postgres struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Postgres {
	return &Postgres{db}
}

func (p *Postgres) Create(ctx context.Context, creatorID int32, title string, description string, startTime time.Time, endTime time.Time, durationMins int32, maxEntries int32, allowLateJoin bool, isDraft bool) (int32, error) {
	var id int32
	var err error

	query := `INSERT INTO contests (creator_id, title, description, start_time, end_time, duration_mins, max_entries, allow_late_join, is_draft) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`
	err = p.db.QueryRowContext(ctx, query, creatorID, title, description, startTime, endTime, durationMins, maxEntries, allowLateJoin, isDraft).Scan(&id)

	return id, err
}

func (p *Postgres) AddProblems(ctx context.Context, contestID int32, problemIDs ...int32) error {
	if len(problemIDs) == 0 {
		return nil
	}

	charcodes := strings.Split("ABCDEFGHIJKLMNOPQRSTUVWXYZ", "")

	if len(problemIDs) > len(charcodes) {
		return fmt.Errorf("not enough charcodes for the number of problems")
	}

	query := `INSERT INTO contest_problems (contest_id, problem_id, charcode) VALUES `
	values := make([]interface{}, 0, len(problemIDs)*3)
	placeholders := make([]string, 0, len(problemIDs))

	for i, problemID := range problemIDs {
		placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3))
		values = append(values, contestID, problemID, charcodes[i])
	}

	query += strings.Join(placeholders, ", ")

	_, err := p.db.ExecContext(ctx, query, values...)
	return err
}

func (p *Postgres) GetByID(ctx context.Context, contestID int32) (*models.Contest, error) {
	var err error
	var contest models.Contest

	query := `SELECT contests.*, users.address AS creator_address, COUNT(entries.id) AS participants
FROM
    contests
JOIN
    users ON users.id = contests.creator_id
LEFT JOIN
    entries ON entries.contest_id = contests.id
WHERE
    contests.id = $1
GROUP BY
    contests.id, users.address`
	err = p.db.GetContext(ctx, &contest, query, contestID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerr.ErrContestNotFound
	}
	if err != nil {
		return nil, err
	}

	return &contest, nil
}

func (p *Postgres) GetProblemset(ctx context.Context, contestID int32) ([]models.Problem, error) {
	var problems []models.Problem

	query := `SELECT cp.charcode, p.*, u.address AS writer_address
		FROM problems p
		JOIN contest_problems cp ON p.id = cp.problem_id
		JOIN users u ON u.id = p.writer_id
		WHERE cp.contest_id = $1`

	err := p.db.SelectContext(ctx, &problems, query, contestID)
	if errors.Is(err, sql.ErrNoRows) {
		return problems, nil
	}
	if err != nil {
		return nil, err
	}

	return problems, nil
}

func (p *Postgres) GetAll(ctx context.Context) ([]models.Contest, error) {
	var err error
	var contests []models.Contest

	query := `SELECT contests.*, users.address AS creator_address, COUNT(entries.id) AS participants
FROM
    contests
JOIN
    users ON users.id = contests.creator_id
LEFT JOIN
    entries ON entries.contest_id = contests.id
GROUP BY
    contests.id, users.address`
	err = p.db.SelectContext(ctx, &contests, query)
	if err != nil {
		return nil, err
	}

	return contests, nil
}

func (p *Postgres) GetWithCreatorID(ctx context.Context, creatorID int32) ([]models.Contest, error) {
	var err error
	var contests []models.Contest

	query := `SELECT contests.*, users.address AS creator_address, COUNT(entries.id) AS participants
FROM
    contests
JOIN
    users ON users.id = contests.creator_id
LEFT JOIN
    entries ON entries.contest_id = contests.id
WHERE
    contests.creator_id = $1
GROUP BY
    contests.id, users.address`
	err = p.db.SelectContext(ctx, &contests, query, creatorID)
	if err != nil {
		return nil, err
	}

	return contests, nil
}

func (p *Postgres) GetEntriesCount(ctx context.Context, contestID int32) (int32, error) {
	var err error
	var count int32

	query := `SELECT COUNT(*) FROM entries WHERE contest_id = $1`
	err = p.db.QueryRowContext(ctx, query, contestID).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (p *Postgres) IsTitleOccupied(ctx context.Context, title string) (bool, error) {
	var err error
	var count int

	query := `SELECT COUNT(*) FROM contests WHERE LOWER(title) = $1`
	err = p.db.QueryRowContext(ctx, query, strings.ToLower(title)).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (p *Postgres) GetLeaderboard(ctx context.Context, contestID int) ([]models.LeaderboardEntry, error) {
	var err error
	var leaderboard []models.LeaderboardEntry

	query := `SELECT
    u.id AS user_id,
    u.address AS user_address,
    SUM(
        CASE
            WHEN p.difficulty = 'easy' THEN 1
            WHEN p.difficulty = 'mid' THEN 3
            WHEN p.difficulty = 'hard' THEN 5
            ELSE 0
        END
    ) AS points
FROM
    users u
JOIN
    entries e ON u.id = e.user_id
JOIN
    contests c ON e.contest_id = c.id
JOIN
    (SELECT DISTINCT entry_id, problem_id
     FROM submissions
     WHERE verdict = 'ok') s ON e.id = s.entry_id
JOIN
    problems p ON s.problem_id = p.id
WHERE
    c.id = 1
GROUP BY
    u.id, u.address
ORDER BY
    points DESC`

	err = p.db.SelectContext(ctx, &leaderboard, query)
	if err != nil {
		return nil, err
	}

	if leaderboard == nil {
		return []models.LeaderboardEntry{}, nil
	}

	return leaderboard, nil
}
