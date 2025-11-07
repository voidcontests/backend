package user

import (
	"context"

	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/internal/storage/repository/postgres"
)

type Postgres struct {
	conn postgres.Transactor
}

func New(conn postgres.Transactor) *Postgres {
	return &Postgres{conn}
}

func (p *Postgres) GetByCredentials(ctx context.Context, username string, passwordHash string) (models.User, error) {
	var user models.User

	query := `SELECT id, username, password_hash, role_id, address, created_at FROM users WHERE username = $1 AND password_hash = $2`
	err := p.conn.QueryRow(ctx, query, username, passwordHash).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.RoleID,
		&user.Address,
		&user.CreatedAt,
	)
	return user, err
}

func (p *Postgres) Create(ctx context.Context, username string, passwordHash string) (models.User, error) {
	var user models.User

	query := `
		INSERT INTO users (username, password_hash, role_id)
		VALUES ($1, $2, (SELECT id FROM roles WHERE is_default = true LIMIT 1))
		RETURNING id, username, password_hash, role_id, address, created_at
	`

	err := p.conn.QueryRow(ctx, query, username, passwordHash).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.RoleID,
		&user.Address,
		&user.CreatedAt,
	)
	return user, err
}

func (p *Postgres) Exists(ctx context.Context, username string) (bool, error) {
	var count int

	query := `SELECT COUNT(*) FROM users WHERE username = $1`
	err := p.conn.QueryRow(ctx, query, username).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (p *Postgres) GetByID(ctx context.Context, id int) (models.User, error) {
	var user models.User

	query := `SELECT id, username, password_hash, role_id, address, created_at FROM users WHERE id = $1`
	err := p.conn.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.RoleID,
		&user.Address,
		&user.CreatedAt,
	)
	return user, err
}

func (p *Postgres) GetByUsername(ctx context.Context, username string) (models.User, error) {
	var user models.User

	query := `SELECT id, username, password_hash, role_id, address, created_at FROM users WHERE username = $1`
	err := p.conn.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.RoleID,
		&user.Address,
		&user.CreatedAt,
	)
	return user, err
}

func (p *Postgres) GetRole(ctx context.Context, userID int) (models.Role, error) {
	var role models.Role

	query := `
		SELECT r.id, r.name, r.created_problems_limit, r.created_contests_limit, r.is_default, r.created_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1
	`
	err := p.conn.QueryRow(ctx, query, userID).Scan(
		&role.ID,
		&role.Name,
		&role.CreatedProblemsLimit,
		&role.CreatedContestsLimit,
		&role.IsDefault,
		&role.CreatedAt,
	)
	return role, err
}

func (p *Postgres) GetCreatedProblemsCount(ctx context.Context, userID int) (int, error) {
	var count int

	query := `SELECT COUNT(*) FROM problems WHERE writer_id = $1`
	err := p.conn.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

func (p *Postgres) GetCreatedContestsCount(ctx context.Context, userID int) (int, error) {
	var count int

	query := `SELECT COUNT(*) FROM contests WHERE creator_id = $1`
	err := p.conn.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

func (p *Postgres) UpdateUser(ctx context.Context, userID int, params models.UpdateUserParams) (models.User, error) {
	var user models.User

	query := `
		UPDATE users
		SET
			username = COALESCE($2, username),
			address = CASE WHEN $3::text IS NOT NULL THEN $3 ELSE address END
		WHERE id = $1
		RETURNING id, username, password_hash, role_id, address, created_at
	`

	err := p.conn.QueryRow(ctx, query, userID, params.Username, params.Address).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.RoleID,
		&user.Address,
		&user.CreatedAt,
	)
	return user, err
}
