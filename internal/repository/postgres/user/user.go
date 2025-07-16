package user

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/voidcontests/backend/internal/repository/models"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool}
}

func (p *Postgres) GetByCredentials(ctx context.Context, username string, passwordHash string) (models.User, error) {
	var user models.User

	query := `SELECT id, username, password_hash, role_id, created_at FROM users WHERE username = $1 AND password_hash = $2`
	err := p.pool.QueryRow(ctx, query, username, passwordHash).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.RoleID,
		&user.CreatedAt,
	)
	return user, err
}

func (p *Postgres) Create(ctx context.Context, username string, passwordHash string) (models.User, error) {
	var user models.User

	query := `
		INSERT INTO users (username, password_hash, role_id)
		VALUES ($1, $2, (SELECT id FROM roles WHERE is_default = true LIMIT 1))
		RETURNING id, username, password_hash, role_id, created_at
	`

	err := p.pool.QueryRow(ctx, query, username, passwordHash).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.RoleID,
		&user.CreatedAt,
	)
	return user, err
}

func (p *Postgres) Exists(ctx context.Context, username string) (bool, error) {
	var count int

	query := `SELECT COUNT(*) FROM users WHERE username = $1`
	err := p.pool.QueryRow(ctx, query, username).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (p *Postgres) GetByID(ctx context.Context, id int32) (models.User, error) {
	var user models.User

	query := `SELECT id, username, password_hash, role_id, created_at FROM users WHERE id = $1`
	err := p.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.RoleID,
		&user.CreatedAt,
	)
	return user, err
}

func (p *Postgres) GetRole(ctx context.Context, userID int32) (models.Role, error) {
	var role models.Role

	query := `
		SELECT r.id, r.name, r.created_problems_limit, r.created_contests_limit, r.is_default, r.created_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1
	`
	err := p.pool.QueryRow(ctx, query, userID).Scan(
		&role.ID,
		&role.Name,
		&role.CreatedProblemsLimit,
		&role.CreatedContestsLimit,
		&role.IsDefault,
		&role.CreatedAt,
	)
	return role, err
}

func (p *Postgres) GetCreatedProblemsCount(ctx context.Context, userID int32) (int, error) {
	var count int

	query := `SELECT COUNT(*) FROM problems WHERE writer_id = $1`
	err := p.pool.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

func (p *Postgres) GetCreatedContestsCount(ctx context.Context, userID int32) (int, error) {
	var count int

	query := `SELECT COUNT(*) FROM contests WHERE creator_id = $1`
	err := p.pool.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}
