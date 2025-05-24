package handler

import (
	"errors"
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
	"github.com/voidcontests/backend/internal/repository/repoerr"
	"github.com/voidcontests/backend/pkg/requestid"
	"github.com/voidcontests/backend/pkg/validate"
)

func (h *Handler) CreateAccount(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.CreateAccount"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	var body request.CreateAccount
	if err := validate.Bind(c, &body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		return Error(http.StatusBadRequest, "invalid body: missing required fields")
	}

	passwordHash := hasher.Sha256String([]byte(body.Password), []byte(h.config.Security.Salt))
	user, err := h.repo.User.Create(ctx, body.Username, passwordHash)
	if err != nil {
		log.Debug("can't create user", sl.Err(err))
		return err
	}

	return c.JSON(http.StatusCreated, response.ID{
		ID: user.ID,
	})
}

func (h *Handler) CreateSession(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.CreateSession"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	var body request.CreateSession
	if err := validate.Bind(c, &body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		return Error(http.StatusBadRequest, "invalid body: missing required fields")
	}

	passwordHash := hasher.Sha256String([]byte(body.Password), []byte(h.config.Security.Salt))
	user, err := h.repo.User.GetByCredentials(ctx, body.Username, passwordHash)
	if errors.Is(err, repoerr.ErrUserNotFound) {
		return Error(http.StatusUnauthorized, "invalid credentials")
	}
	if err != nil {
		log.Error("can't create user", sl.Err(err))
		return err
	}

	token, err := jwt.GenerateToken(user.ID, h.config.Security.SignatureKey)
	if err != nil {
		slog.Error("jwt: can't generate token", sl.Err(err))
		return err
	}

	return c.JSON(http.StatusCreated, response.Token{
		Token: token,
	})
}

func (h *Handler) GetAccount(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.GetAccount"), slog.String("request_id", requestid.Get(c)))
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	user, err := h.repo.User.GetByID(ctx, claims.UserID)
	if err != nil {
		log.Debug("can't get user", sl.Err(err))
		return err
	}

	role, err := h.repo.User.GetRole(ctx, claims.UserID)
	if err != nil {
		log.Debug("can't get user role", sl.Err(err))
		return err
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
