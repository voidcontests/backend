package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	repoerr "github.com/voidcontests/backend/internal/repository/errors"
	"github.com/voidcontests/backend/internal/repository/models"
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

	query := `INSERT INTO users (address) VALUES ($1) RETURNING *`
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

func (p *Postgres) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var err error
	var user models.User

	query := `SELECT * FROM users WHERE username = $1`
	err = p.db.GetContext(ctx, &user, query, username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repoerr.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}
