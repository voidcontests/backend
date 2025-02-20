package user

import (
	"context"
	"database/sql"
	"errors"

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

func (p *Postgres) Create(ctx context.Context, address string) (*models.User, error) {
	var err error
	var user models.User

	query := `INSERT INTO users (address, role_id) VALUES ($1, (SELECT id FROM roles WHERE is_default=true LIMIT 1)) RETURNING id`
	err = p.db.GetContext(ctx, &user, query, address)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (p *Postgres) GetByAddress(ctx context.Context, address string) (*models.User, error) {
	var err error
	var user models.User

	query := `SELECT * FROM users WHERE address = $1`
	err = p.db.GetContext(ctx, &user, query, address)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerr.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (p *Postgres) GetRole(ctx context.Context, userID int32) (*models.Role, error) {
	var err error
	var role models.Role

	query := `SELECT r.* FROM users u JOIN roles r ON u.role_id = r.id WHERE u.id = $1`
	err = p.db.GetContext(ctx, &role, query, userID)
	if err != nil {
		return nil, err
	}

	return &role, nil
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
