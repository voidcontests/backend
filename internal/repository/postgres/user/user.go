package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/voidcontests/backend/internal/repository/models"
)

type Postgres struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Postgres {
	return &Postgres{db}
}

func (p *Postgres) GetByCredentials(ctx context.Context, username string, passwordHash string) (models.User, error) {
	var user models.User

	query := `SELECT * FROM users WHERE username = $1 AND password_hash = $2`
	err := p.db.GetContext(ctx, &user, query, username, passwordHash)
	return user, err
}

func (p *Postgres) Create(ctx context.Context, username string, passwordHash string) (models.User, error) {
	var user models.User

	query := `INSERT INTO users (username, password_hash, role_id) VALUES ($1, $2, (SELECT id FROM roles WHERE is_default=true LIMIT 1)) RETURNING id`
	err := p.db.GetContext(ctx, &user, query, username, passwordHash)
	return user, err
}

func (p *Postgres) Exists(ctx context.Context, username string) (bool, error) {
	var err error
	var count uint

	query := `SELECT COUNT(*) FROM users WHERE username = $1`
	err = p.db.GetContext(ctx, &count, query, username)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (p *Postgres) GetByID(ctx context.Context, id int32) (models.User, error) {
	var user models.User

	query := `SELECT * FROM users WHERE id = $1`
	err := p.db.GetContext(ctx, &user, query, id)
	return user, err
}

func (p *Postgres) GetRole(ctx context.Context, userID int32) (models.Role, error) {
	var role models.Role

	query := `SELECT r.* FROM users u JOIN roles r ON u.role_id = r.id WHERE u.id = $1`
	err := p.db.GetContext(ctx, &role, query, userID)
	return role, err
}

func (p *Postgres) GetCreatedProblemsCount(ctx context.Context, userID int32) (int, error) {
	var err error
	var count int

	query := `SELECT COUNT(*) FROM problems WHERE writer_id = $1`
	err = p.db.GetContext(ctx, &count, query, userID)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (p *Postgres) GetCreatedContestsCount(ctx context.Context, userID int32) (int, error) {
	var err error
	var count int

	query := `SELECT COUNT(*) FROM contests WHERE creator_id = $1`
	err = p.db.GetContext(ctx, &count, query, userID)
	if err != nil {
		return 0, err
	}

	return count, nil
}
