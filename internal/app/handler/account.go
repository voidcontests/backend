package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	jwtgo "github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/dto/request"
	"github.com/voidcontests/backend/internal/app/handler/dto/response"
	"github.com/voidcontests/backend/internal/hasher"
	"github.com/voidcontests/backend/internal/jwt"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/pkg/requestid"
	"github.com/voidcontests/backend/pkg/validate"
)

func (h *Handler) CreateAccount(c echo.Context) error {
	op := "handler.CreateAccount"
	ctx := c.Request().Context()

	var body request.CreateAccount
	if err := validate.Bind(c, &body); err != nil {
		return Error(http.StatusBadRequest, "invalid body: missing required fields")
	}

	exists, err := h.repo.User.Exists(ctx, body.Username)
	if err != nil {
		return fmt.Errorf("%s: can't verify that user exists or not: %v", op, err)
	}

	if exists {
		return Error(http.StatusConflict, "user already exists")
	}

	passwordHash := hasher.Sha256String([]byte(body.Password), []byte(h.config.Security.Salt))
	user, err := h.repo.User.Create(ctx, body.Username, passwordHash)
	if err != nil {
		return fmt.Errorf("%s: failed to create user: %v", op, err)
	}

	return c.JSON(http.StatusCreated, response.ID{
		ID: user.ID,
	})
}

func (h *Handler) CreateSession(c echo.Context) error {
	op := "handler.CreateSession"
	ctx := c.Request().Context()

	var body request.CreateSession
	if err := validate.Bind(c, &body); err != nil {
		return Error(http.StatusBadRequest, "invalid body: missing required fields")
	}

	passwordHash := hasher.Sha256String([]byte(body.Password), []byte(h.config.Security.Salt))
	user, err := h.repo.User.GetByCredentials(ctx, body.Username, passwordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return Error(http.StatusUnauthorized, "user not found")
	}
	if err != nil {
		return fmt.Errorf("%s: can't create user: %v", op, err)
	}

	token, err := jwt.GenerateToken(user.ID, h.config.Security.SignatureKey)
	if err != nil {
		return fmt.Errorf("%s: can't generate token: %v", op, err)
	}

	return c.JSON(http.StatusCreated, response.Token{
		Token: token,
	})
}

func (h *Handler) GetAccount(c echo.Context) error {
	op := "handler.GetAccount"
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	user, err := h.repo.User.GetByID(ctx, claims.UserID)
	if errors.Is(err, sql.ErrNoRows) {
		return Error(http.StatusNotFound, "user not found")
	}
	if err != nil {
		return fmt.Errorf("%s: can't get user: %v", op, err)
	}

	role, err := h.repo.User.GetRole(ctx, claims.UserID)
	if err != nil {
		return fmt.Errorf("%s: can't get role: %v", op, err)
	}

	return c.JSON(http.StatusOK, response.Account{
		ID:       user.ID,
		Username: user.Username,
		Role: response.Role{
			Name:                 role.Name,
			CreatedProblemsLimit: role.CreatedProblemsLimit,
			CreatedContestsLimit: role.CreatedContestsLimit,
		},
	})
}

func (h *Handler) TryIdentify() echo.MiddlewareFunc {
	return h.UserIdentity(true)
}

func (h *Handler) MustIdentify() echo.MiddlewareFunc {
	return h.UserIdentity(false)
}

func (h *Handler) UserIdentity(skiperr bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			log := slog.With(slog.String("op", "handler.UserIdentify"), slog.String("request_id", requestid.Get(c)))

			authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
			if authHeader == "" {
				log.Debug("auth header is empty, skipping check")
				if skiperr {
					return next(c)
				} else {
					return Error(http.StatusUnauthorized, "invalid or malformed token")
				}
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				log.Debug("invalid auth header format, skipping check")
				if skiperr {
					return next(c)
				} else {
					return Error(http.StatusUnauthorized, "invalid or malformed token")
				}
			}

			tokenString := parts[1]

			token, err := jwtgo.ParseWithClaims(tokenString, &jwt.CustomClaims{}, func(token *jwtgo.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwtgo.SigningMethodHMAC); !ok {
					return nil, echo.NewHTTPError(http.StatusUnauthorized, "unexpected signing method")
				}
				return []byte(h.config.Security.SignatureKey), nil
			})

			if err != nil {
				log.Debug("token parsing failed", sl.Err(err))
				if skiperr {
					return next(c)
				} else {
					return Error(http.StatusUnauthorized, "invalid or malformed token")
				}
			}

			if !token.Valid {
				log.Debug("invalid token")
				if skiperr {
					return next(c)
				} else {
					return Error(http.StatusUnauthorized, "invalid or malformed token")
				}
			}

			claims, ok := token.Claims.(*jwt.CustomClaims)
			if !ok {
				log.Debug("invalid token claims")
				if skiperr {
					return next(c)
				} else {
					return Error(http.StatusUnauthorized, "invalid or malformed token")
				}
			}

			c.Set("account", *claims)

			return next(c)
		}
	}
}
