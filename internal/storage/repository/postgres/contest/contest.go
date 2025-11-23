package contest

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/internal/storage/repository/postgres"
)

const defaultLimit = 20

type Postgres struct {
	conn postgres.Transactor
}

func New(txr postgres.Transactor) *Postgres {
	return &Postgres{conn: txr}
}

func (p *Postgres) Create(ctx context.Context, creatorID int, title, desc, awardType string, entryPriceTonNanos uint64, startTime, endTime time.Time, durationMins, maxEntries int, allowLateJoin bool, problems []models.ProblemCharcode, walletID *int) (int, error) {
	var contestID int
	var err error

	err = p.conn.QueryRow(ctx, `
INSERT INTO contests (creator_id, title, description, award_type, entry_price_ton_nanos, start_time, end_time, duration_mins, max_entries, allow_late_join, wallet_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id
`, creatorID, title, desc, awardType, entryPriceTonNanos, startTime, endTime, durationMins, maxEntries, allowLateJoin, walletID).Scan(&contestID)

	if err != nil {
		return 0, fmt.Errorf("insert contest failed: %w", err)
	}

	if len(problems) > 0 {
		batch := &pgx.Batch{}
		for _, p := range problems {
			batch.Queue(`INSERT INTO contest_problems (contest_id, problem_id, charcode) VALUES ($1, $2, $3)`,
				contestID, p.ProblemID, p.Charcode)
		}

		br := p.conn.SendBatch(ctx, batch)

		for i := 0; i < len(problems); i++ {
			if _, err := br.Exec(); err != nil {
				br.Close()
				return 0, fmt.Errorf("insert contest_problem %d (problem_id=%d, charcode=%s) failed: %w", i, problems[i].ProblemID, problems[i].Charcode, err)
			}
		}

		if err := br.Close(); err != nil {
			return 0, fmt.Errorf("batch close failed: %w", err)
		}
	}

	return contestID, nil
}

func (p *Postgres) GetByID(ctx context.Context, contestID int) (models.Contest, error) {
	var contest models.Contest
	query := `
SELECT
	id, creator_id, creator_username, creator_address, title, description, award_type, entry_price_ton_nanos, start_time, end_time, duration_mins,
	max_entries, allow_late_join, wallet_id, distribution_payment_id, participants_count, created_at
FROM contests_view
WHERE id = $1`
	err := p.conn.QueryRow(ctx, query, contestID).Scan(
		&contest.ID, &contest.CreatorID, &contest.CreatorUsername, &contest.CreatorAddress, &contest.Title, &contest.Description, &contest.AwardType, &contest.EntryPriceTonNanos, &contest.StartTime,
		&contest.EndTime, &contest.DurationMins, &contest.MaxEntries, &contest.AllowLateJoin,
		&contest.WalletID, &contest.DistributionPaymentID, &contest.ParticipantsCount, &contest.CreatedAt)
	return contest, err
}

func (p *Postgres) GetWallet(ctx context.Context, walletID int) (models.Wallet, error) {
	var wallet models.Wallet
	query := `
SELECT
	w.id, w.address, w.mnemonic_encrypted, w.created_at
FROM wallets w
WHERE w.id = $1`
	err := p.conn.QueryRow(ctx, query, walletID).Scan(&wallet.ID, &wallet.Address, &wallet.MnemonicEncrypted, &wallet.CreatedAt)
	return wallet, err
}

func (p *Postgres) GetProblemset(ctx context.Context, contestID int) ([]models.Problem, error) {
	query := `
SELECT
	problem_id, charcode, writer_id, writer_username, writer_address, title, statement,
	difficulty, time_limit_ms, memory_limit_mb, checker, created_at
FROM contest_problems_view
WHERE contest_id = $1 ORDER BY charcode ASC`

	rows, err := p.conn.Query(ctx, query, contestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var problems []models.Problem
	for rows.Next() {
		var problem models.Problem
		if err := rows.Scan(&problem.ID, &problem.Charcode, &problem.WriterID, &problem.WriterUsername, &problem.WriterAddress, &problem.Title, &problem.Statement, &problem.Difficulty, &problem.TimeLimitMS, &problem.MemoryLimitMB, &problem.Checker, &problem.CreatedAt); err != nil {
			return nil, err
		}
		problems = append(problems, problem)
	}
	return problems, nil
}

func (p *Postgres) ListAll(ctx context.Context, limit int, offset int, filters models.ContestFilters) (contests []models.Contest, total int, err error) {
	if limit < 0 {
		limit = defaultLimit
	}

	batch := &pgx.Batch{}

	whereClauses := []string{"end_time >= now()"}
	queryArgs := []interface{}{limit, offset}
	countArgs := []interface{}{}
	paramIndex := 3

	if filters.CreatorID != 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("creator_id = $%d", paramIndex))
		queryArgs = append(queryArgs, filters.CreatorID)
		countArgs = append(countArgs, filters.CreatorID)
		paramIndex++
	}

	if filters.Title != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("LOWER(title) LIKE LOWER($%d)", paramIndex))
		queryArgs = append(queryArgs, "%"+filters.Title+"%")
		countArgs = append(countArgs, "%"+filters.Title+"%")
		paramIndex++
	}

	whereClause := "WHERE " + strings.Join(whereClauses, " AND ")

	query := fmt.Sprintf(`
SELECT
	id, creator_id, creator_username, creator_address, title, description, award_type, entry_price_ton_nanos, start_time, end_time, duration_mins, max_entries,
	allow_late_join, wallet_id, distribution_payment_id, participants_count, created_at
FROM contests_view
%s
ORDER BY id ASC
LIMIT $1 OFFSET $2
	`, whereClause)

	batch.Queue(query, queryArgs...)

	countWhereClauses := []string{"end_time >= now()"}
	if filters.CreatorID != 0 {
		countWhereClauses = append(countWhereClauses, "creator_id = $1")
	}
	if filters.Title != "" {
		countParamIndex := 1
		if filters.CreatorID != 0 {
			countParamIndex = 2
		}
		countWhereClauses = append(countWhereClauses, fmt.Sprintf("LOWER(title) LIKE LOWER($%d)", countParamIndex))
	}
	countWhereClause := strings.Join(countWhereClauses, " AND ")
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM contests_view WHERE %s", countWhereClause)

	if len(countArgs) > 0 {
		batch.Queue(countQuery, countArgs...)
	} else {
		batch.Queue(countQuery)
	}

	br := p.conn.SendBatch(ctx, batch)

	rows, err := br.Query()
	if err != nil {
		br.Close()
		return nil, 0, fmt.Errorf("contests query failed: %w", err)
	}

	contests = make([]models.Contest, 0)
	for rows.Next() {
		var c models.Contest
		if err := rows.Scan(
			&c.ID, &c.CreatorID, &c.CreatorUsername, &c.CreatorAddress, &c.Title, &c.Description, &c.AwardType, &c.EntryPriceTonNanos,
			&c.StartTime, &c.EndTime, &c.DurationMins,
			&c.MaxEntries, &c.AllowLateJoin, &c.WalletID, &c.DistributionPaymentID, &c.ParticipantsCount, &c.CreatedAt,
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

func (p *Postgres) GetWithCreatorID(ctx context.Context, creatorID int, limit, offset int) (contests []models.Contest, total int, err error) {
	batch := &pgx.Batch{}
	batch.Queue(`
SELECT
	id, creator_id, creator_username, creator_address, title, description, award_type, entry_price_ton_nanos, start_time, end_time, duration_mins, max_entries,
	allow_late_join, wallet_id, distribution_payment_id, participants_count, created_at
FROM contests_view
WHERE creator_id = $1
ORDER BY id ASC
LIMIT $2 OFFSET $3
	`, creatorID, limit, offset)

	batch.Queue(`SELECT COUNT(*) FROM contests_view WHERE creator_id = $1`, creatorID)

	br := p.conn.SendBatch(ctx, batch)

	rows, err := br.Query()
	if err != nil {
		br.Close()
		return nil, 0, err
	}

	contests = make([]models.Contest, 0)
	for rows.Next() {
		var c models.Contest
		if err := rows.Scan(
			&c.ID, &c.CreatorID, &c.CreatorUsername, &c.CreatorAddress, &c.Title, &c.Description, &c.AwardType, &c.EntryPriceTonNanos,
			&c.StartTime, &c.EndTime, &c.DurationMins,
			&c.MaxEntries, &c.AllowLateJoin, &c.WalletID, &c.DistributionPaymentID, &c.ParticipantsCount, &c.CreatedAt,
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

func (p *Postgres) GetEntriesCount(ctx context.Context, contestID int) (int, error) {
	var count int
	err := p.conn.QueryRow(ctx, `SELECT COUNT(*) FROM entries WHERE contest_id = $1`, contestID).Scan(&count)
	return count, err
}

func (p *Postgres) IsTitleOccupied(ctx context.Context, title string) (bool, error) {
	var count int
	err := p.conn.QueryRow(ctx, `SELECT COUNT(*) FROM contests WHERE LOWER(title) = $1`, strings.ToLower(title)).Scan(&count)
	return count > 0, err
}

func (p *Postgres) GetWinnerID(ctx context.Context, contestID int) (int, error) {
	// TODO: tie-breaking rules
	query := ` SELECT user_id FROM scores
		WHERE contest_id = $1 AND points > 0 ORDER BY points DESC LIMIT 1`

	var userID int
	err := p.conn.QueryRow(ctx, query, contestID).Scan(&userID)
	return userID, err
}

func (p *Postgres) GetScores(ctx context.Context, contestID, limit, offset int) (scores []models.ScoresEntry, total int, err error) {
	query := `
		SELECT user_id, username, points, COUNT(*) OVER() AS total
		FROM scores
		WHERE contest_id = $1
		ORDER BY points DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := p.conn.Query(ctx, query, contestID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("scores query failed: %w", err)
	}
	defer rows.Close()

	scores = make([]models.ScoresEntry, 0)
	for rows.Next() {
		var entry models.ScoresEntry
		if err := rows.Scan(&entry.UserID, &entry.Username, &entry.Points, &total); err != nil {
			return nil, 0, err
		}
		scores = append(scores, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return scores, total, nil
}

func (p *Postgres) SetDistributionPaymentID(ctx context.Context, contestID int, paymentID int) error {
	query := `UPDATE contests SET distribution_payment_id = $1 WHERE id = $2`
	_, err := p.conn.Exec(ctx, query, paymentID, contestID)
	if err != nil {
		return fmt.Errorf("set distribution_payment_id failed: %w", err)
	}
	return nil
}

func (p *Postgres) GetWithUndistributedAwards(ctx context.Context) ([]models.Contest, error) {
	query := `
SELECT
	id, creator_id, creator_username, creator_address, title, description, award_type, entry_price_ton_nanos, start_time, end_time, duration_mins,
	max_entries, allow_late_join, wallet_id, distribution_payment_id, participants_count, created_at
FROM contests_view
WHERE distribution_payment_id IS NULL AND end_time < now() AND award_type <> 'no'
ORDER BY end_time ASC`

	rows, err := p.conn.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query contests with undistributed awards failed: %w", err)
	}
	defer rows.Close()

	contests := make([]models.Contest, 0)
	for rows.Next() {
		var c models.Contest
		if err := rows.Scan(
			&c.ID, &c.CreatorID, &c.CreatorUsername, &c.CreatorAddress, &c.Title, &c.Description, &c.AwardType, &c.EntryPriceTonNanos,
			&c.StartTime, &c.EndTime, &c.DurationMins,
			&c.MaxEntries, &c.AllowLateJoin, &c.WalletID, &c.DistributionPaymentID, &c.ParticipantsCount, &c.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		contests = append(contests, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration failed: %w", err)
	}

	return contests, nil
}
