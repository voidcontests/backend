package contest

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/voidcontests/backend/internal/repository/models"
)

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

func (p *Postgres) AddProblems(ctx context.Context, contestID int32, problemIDs ...int32) error {
	if len(problemIDs) == 0 {
		return nil
	}
	charcodes := strings.Split("ABCDEFGHIJKLMNOPQRSTUVWXYZ", "")
	if len(problemIDs) > len(charcodes) {
		return fmt.Errorf("not enough charcodes for the number of problems")
	}

	query := `INSERT INTO contest_problems (contest_id, problem_id, charcode) VALUES `
	placeholders := make([]string, len(problemIDs))
	args := make([]interface{}, len(problemIDs)*3)
	for i, id := range problemIDs {
		placeholders[i] = fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3)
		args[i*3], args[i*3+1], args[i*3+2] = contestID, id, charcodes[i]
	}
	query += strings.Join(placeholders, ", ")
	_, err := p.pool.Exec(ctx, query, args...)
	return err
}

func (p *Postgres) CreateWithProblemIDs(ctx context.Context, creatorID int32, title, desc string, startTime, endTime time.Time, durationMins, maxEntries int32, allowLateJoin bool, problemIDs []int32) (int32, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var contestID int32
	query := `INSERT INTO contests (creator_id, title, description, start_time, end_time, duration_mins, max_entries, allow_late_join)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	err = tx.QueryRow(ctx, query, creatorID, title, desc, startTime, endTime, durationMins, maxEntries, allowLateJoin).Scan(&contestID)
	if err != nil {
		return 0, err
	}

	if len(problemIDs) > 0 {
		charcodes := strings.Split("ABCDEFGHIJKLMNOPQRSTUVWXYZ", "")
		if len(problemIDs) > len(charcodes) {
			return 0, fmt.Errorf("not enough charcodes for the number of problems")
		}
		query = `INSERT INTO contest_problems (contest_id, problem_id, charcode) VALUES `
		placeholders := make([]string, len(problemIDs))
		args := make([]interface{}, len(problemIDs)*3)
		for i, pid := range problemIDs {
			placeholders[i] = fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3)
			args[i*3], args[i*3+1], args[i*3+2] = contestID, pid, charcodes[i]
		}
		query += strings.Join(placeholders, ", ")
		_, err = tx.Exec(ctx, query, args...)
		if err != nil {
			return 0, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
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

func (p *Postgres) GetAll(ctx context.Context) ([]models.Contest, error) {
	query := `SELECT contests.*, users.username AS creator_username, COUNT(entries.id) AS participants
		FROM contests
		JOIN users ON users.id = contests.creator_id
		LEFT JOIN entries ON entries.contest_id = contests.id
		GROUP BY contests.id, users.username
		ORDER BY contests.id ASC`
	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contests []models.Contest
	for rows.Next() {
		var c models.Contest
		if err := rows.Scan(&c.ID, &c.CreatorID, &c.Title, &c.Description, &c.StartTime, &c.EndTime, &c.DurationMins, &c.MaxEntries, &c.AllowLateJoin, &c.CreatedAt, &c.CreatorUsername, &c.Participants); err != nil {
			return nil, err
		}
		contests = append(contests, c)
	}
	return contests, nil
}

func (p *Postgres) GetWithCreatorID(ctx context.Context, creatorID int32) ([]models.Contest, error) {
	query := `SELECT contests.*, users.username AS creator_username, COUNT(entries.id) AS participants
		FROM contests
		JOIN users ON users.id = contests.creator_id
		LEFT JOIN entries ON entries.contest_id = contests.id
		WHERE contests.creator_id = $1
		GROUP BY contests.id, users.username`

	rows, err := p.pool.Query(ctx, query, creatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contests []models.Contest
	for rows.Next() {
		var c models.Contest
		if err := rows.Scan(&c.ID, &c.CreatorID, &c.Title, &c.Description, &c.StartTime, &c.EndTime, &c.DurationMins, &c.MaxEntries, &c.AllowLateJoin, &c.CreatedAt, &c.CreatorUsername, &c.Participants); err != nil {
			return nil, err
		}
		contests = append(contests, c)
	}
	return contests, nil
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

func (p *Postgres) GetLeaderboard(ctx context.Context, contestID int) ([]models.LeaderboardEntry, error) {
	query := `SELECT u.id AS user_id, u.username, COALESCE(SUM(
		CASE WHEN p.difficulty = 'easy' THEN 1
			 WHEN p.difficulty = 'mid' THEN 3
			 WHEN p.difficulty = 'hard' THEN 5 ELSE 0 END), 0) AS points
		FROM users u
		JOIN entries e ON u.id = e.user_id
		JOIN contests c ON e.contest_id = c.id
		LEFT JOIN (SELECT DISTINCT entry_id, problem_id FROM submissions WHERE verdict = 'ok') s ON e.id = s.entry_id
		LEFT JOIN problems p ON s.problem_id = p.id
		WHERE c.id = $1
		GROUP BY u.id, u.username
		ORDER BY points DESC`

	rows, err := p.pool.Query(ctx, query, contestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leaderboard []models.LeaderboardEntry
	for rows.Next() {
		var entry models.LeaderboardEntry
		if err := rows.Scan(&entry.UserID, &entry.Username, &entry.Points); err != nil {
			return nil, err
		}
		leaderboard = append(leaderboard, entry)
	}
	return leaderboard, nil
}
