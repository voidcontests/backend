package contest

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/voidcontests/backend/internal/repository/models"
)

const defaultLimit = 20

type Postgres struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool}
}

func (p *Postgres) Create(ctx context.Context, creatorID int32, title, description string, startTime, endTime time.Time, durationMins, maxEntries int32, allowLateJoin bool) (int32, error) {
	var id int32
	query := `INSERT INTO contests (creator_id, title, description, start_time, end_time, duration_mins, max_entries, allow_late_join)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	err := p.pool.QueryRow(ctx, query, creatorID, title, description, startTime, endTime, durationMins, maxEntries, allowLateJoin).Scan(&id)
	return id, err
}

func (p *Postgres) CreateWithProblemIDs(ctx context.Context, creatorID int32, title, desc string, startTime, endTime time.Time, durationMins, maxEntries int32, allowLateJoin bool, problemIDs []int32) (int32, error) {
	charcodes := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	if len(problemIDs) > len(charcodes) {
		return 0, fmt.Errorf("not enough charcodes for the number of problems")
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var contestID int32
	err = tx.QueryRow(ctx, `
		INSERT INTO contests
		(creator_id, title, description, start_time, end_time, duration_mins, max_entries, allow_late_join)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, creatorID, title, desc, startTime, endTime, durationMins, maxEntries, allowLateJoin).Scan(&contestID)
	if err != nil {
		return 0, fmt.Errorf("insert contest failed: %w", err)
	}

	batch := &pgx.Batch{}
	for i, pid := range problemIDs {
		batch.Queue(`
			INSERT INTO contest_problems (contest_id, problem_id, charcode)
			VALUES ($1, $2, $3)
		`, contestID, pid, string(charcodes[i]))
	}

	br := tx.SendBatch(ctx, batch)

	for i := 0; i < len(problemIDs); i++ {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return 0, fmt.Errorf("insert contest_problem %d failed: %w", i, err)
		}
	}

	if err := br.Close(); err != nil {
		return 0, fmt.Errorf("batch close failed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit failed: %w", err)
	}

	return contestID, nil
}

func (p *Postgres) GetByID(ctx context.Context, contestID int32) (*models.Contest, error) {
	var contest models.Contest
	query := `SELECT contests.*, users.username AS creator_username, COUNT(entries.id) AS participants
		FROM contests
		JOIN users ON users.id = contests.creator_id
		LEFT JOIN entries ON entries.contest_id = contests.id
		WHERE contests.id = $1
		GROUP BY contests.id, users.username`
	err := p.pool.QueryRow(ctx, query, contestID).Scan(&contest.ID, &contest.CreatorID, &contest.Title, &contest.Description, &contest.StartTime, &contest.EndTime, &contest.DurationMins, &contest.MaxEntries, &contest.AllowLateJoin, &contest.CreatedAt, &contest.CreatorUsername, &contest.Participants)
	if err != nil {
		return nil, err
	}
	return &contest, nil
}

func (p *Postgres) GetProblemset(ctx context.Context, contestID int32) ([]models.Problem, error) {
	query := `SELECT cp.charcode, p.*, u.username AS writer_username
		FROM problems p
		JOIN contest_problems cp ON p.id = cp.problem_id
		JOIN users u ON u.id = p.writer_id
		WHERE cp.contest_id = $1 ORDER BY charcode ASC`

	rows, err := p.pool.Query(ctx, query, contestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var problems []models.Problem
	for rows.Next() {
		var problem models.Problem
		if err := rows.Scan(&problem.Charcode, &problem.ID, &problem.Kind, &problem.WriterID, &problem.Title, &problem.Statement, &problem.Difficulty, &problem.Answer, &problem.TimeLimitMS, &problem.CreatedAt, &problem.WriterUsername); err != nil {
			return nil, err
		}
		problems = append(problems, problem)
	}
	return problems, nil
}

func (p *Postgres) ListAll(ctx context.Context, limit int, offset int) (contests []models.Contest, total int, err error) {
	if limit < 0 {
		limit = defaultLimit
	}

	batch := &pgx.Batch{}
	batch.Queue(`
		SELECT contests.*, users.username AS creator_username, COUNT(entries.id) AS participants
		FROM contests
		JOIN users ON users.id = contests.creator_id
		LEFT JOIN entries ON entries.contest_id = contests.id
		WHERE contests.end_time >= now()
		GROUP BY contests.id, users.username
		ORDER BY contests.id ASC
		LIMIT $1 OFFSET $2
	`, limit, offset)

	batch.Queue(`SELECT COUNT(*) FROM contests WHERE contests.end_time >= now()`)

	br := p.pool.SendBatch(ctx, batch)

	rows, err := br.Query()
	if err != nil {
		br.Close()
		return nil, 0, fmt.Errorf("contests query failed: %w", err)
	}

	for rows.Next() {
		var c models.Contest
		if err := rows.Scan(
			&c.ID, &c.CreatorID, &c.Title, &c.Description,
			&c.StartTime, &c.EndTime, &c.DurationMins,
			&c.MaxEntries, &c.AllowLateJoin, &c.CreatedAt,
			&c.CreatorUsername, &c.Participants,
		); err != nil {
			rows.Close()
			br.Close()
			return nil, 0, fmt.Errorf("scan failed: %w", err)
		}
		contests = append(contests, c)
	}
	rows.Close()

	if err := br.QueryRow().Scan(&total); err != nil {
		br.Close()
		return nil, 0, fmt.Errorf("count query failed: %w", err)
	}

	if err := br.Close(); err != nil {
		return nil, 0, fmt.Errorf("batch close failed: %w", err)
	}

	return contests, total, nil
}

func (p *Postgres) GetWithCreatorID(ctx context.Context, creatorID int32, limit, offset int) (contests []models.Contest, total int, err error) {
	batch := &pgx.Batch{}
	batch.Queue(`
		SELECT contests.*, users.username AS creator_username, COUNT(entries.id) AS participants
		FROM contests
		JOIN users ON users.id = contests.creator_id
		LEFT JOIN entries ON entries.contest_id = contests.id
		WHERE contests.creator_id = $1
		GROUP BY contests.id, users.username
		ORDER BY contests.id ASC
		LIMIT $2 OFFSET $3
	`, creatorID, limit, offset)

	batch.Queue(`SELECT COUNT(*) FROM contests WHERE creator_id = $1`, creatorID)

	br := p.pool.SendBatch(ctx, batch)

	rows, err := br.Query()
	if err != nil {
		br.Close()
		return nil, 0, err
	}

	for rows.Next() {
		var c models.Contest
		if err := rows.Scan(
			&c.ID, &c.CreatorID, &c.Title, &c.Description,
			&c.StartTime, &c.EndTime, &c.DurationMins,
			&c.MaxEntries, &c.AllowLateJoin, &c.CreatedAt,
			&c.CreatorUsername, &c.Participants,
		); err != nil {
			rows.Close()
			br.Close()
			return nil, 0, err
		}
		contests = append(contests, c)
	}
	rows.Close()

	if err := br.QueryRow().Scan(&total); err != nil {
		br.Close()
		return nil, 0, err
	}

	if err := br.Close(); err != nil {
		return nil, 0, err
	}

	return contests, total, nil
}

func (p *Postgres) GetEntriesCount(ctx context.Context, contestID int32) (int32, error) {
	var count int32
	err := p.pool.QueryRow(ctx, `SELECT COUNT(*) FROM entries WHERE contest_id = $1`, contestID).Scan(&count)
	return count, err
}

func (p *Postgres) IsTitleOccupied(ctx context.Context, title string) (bool, error) {
	var count int
	err := p.pool.QueryRow(ctx, `SELECT COUNT(*) FROM contests WHERE LOWER(title) = $1`, strings.ToLower(title)).Scan(&count)
	return count > 0, err
}

func (p *Postgres) GetLeaderboard(ctx context.Context, contestID, limit, offset int) (leaderboard []models.LeaderboardEntry, total int, err error) {
	batch := &pgx.Batch{}
	batch.Queue(`
		SELECT u.id AS user_id, u.username, COALESCE(SUM(
			CASE
				WHEN p.difficulty = 'easy' THEN 1
				WHEN p.difficulty = 'mid' THEN 3
				WHEN p.difficulty = 'hard' THEN 5
				ELSE 0
			END
		), 0) AS points
		FROM users u
		JOIN entries e ON u.id = e.user_id
		JOIN contests c ON e.contest_id = c.id
		LEFT JOIN (
			SELECT DISTINCT entry_id, problem_id
			FROM submissions
			WHERE verdict = 'ok'
		) s ON e.id = s.entry_id
		LEFT JOIN problems p ON s.problem_id = p.id
		WHERE c.id = $1
		GROUP BY u.id, u.username
		ORDER BY points DESC
		LIMIT $2 OFFSET $3
	`, contestID, limit, offset)

	batch.Queue(`
		SELECT COUNT(DISTINCT u.id)
		FROM users u
		JOIN entries e ON u.id = e.user_id
		WHERE e.contest_id = $1
	`, contestID)

	br := p.pool.SendBatch(ctx, batch)

	rows, err := br.Query()
	if err != nil {
		br.Close()
		return nil, 0, fmt.Errorf("leaderboard query failed: %w", err)
	}

	for rows.Next() {
		var entry models.LeaderboardEntry
		if err := rows.Scan(&entry.UserID, &entry.Username, &entry.Points); err != nil {
			rows.Close()
			br.Close()
			return nil, 0, err
		}
		leaderboard = append(leaderboard, entry)
	}
	rows.Close()

	if leaderboard == nil {
		leaderboard = make([]models.LeaderboardEntry, 0)
	}

	if err := br.QueryRow().Scan(&total); err != nil {
		br.Close()
		return nil, 0, fmt.Errorf("total count query failed: %w", err)
	}

	if err := br.Close(); err != nil {
		return nil, 0, err
	}

	return leaderboard, total, nil
}
