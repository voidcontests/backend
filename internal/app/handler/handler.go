package handler

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	jwtgo "github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/service/account"
	"github.com/voidcontests/backend/internal/app/service/contest"
	"github.com/voidcontests/backend/internal/app/service/entry"
	"github.com/voidcontests/backend/internal/config"
	"github.com/voidcontests/backend/internal/jwt"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/internal/repository"
	"github.com/voidcontests/backend/pkg/requestid"
)

type Handler struct {
	config         *config.Config
	repo           *repository.Repository
	accountService *account.Service
	entryService   *entry.Service
	contestService *contest.Service
}

func New(c *config.Config, r *repository.Repository) *Handler {
	as := account.NewService(c, r)
	es := entry.NewService(c, r)
	cs := contest.NewService(c, r)

	return &Handler{
		config:         c,
		repo:           r,
		accountService: as,
		entryService:   es,
		contestService: cs,
	}
}

func (h *Handler) Healthcheck(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

func ExtractClaims(c echo.Context) (jwt.CustomClaims, bool) {
	raw := c.Get("account")
	if raw == nil {
		return jwt.CustomClaims{}, false
	}
	claims, ok := raw.(jwt.CustomClaims)
	if !ok {
		return jwt.CustomClaims{}, false
	}
	return claims, true
}

func ExtractQueryParamInt(c echo.Context, key string) (int, bool) {
	param := c.QueryParam(key)
	if param == "" {
		return 0, false
	}

	value, err := strconv.Atoi(param)
	if err != nil {
		return 0, false
	}

	return value, true
}

func ExtractParamInt(c echo.Context, key string) (int, bool) {
	param := c.Param(key)
	value, err := strconv.Atoi(param)
	return value, err == nil
}

type APIError struct {
	Status  int
	Message string
}

func Error(code int, message string) error {
	return &APIError{
		Status:  code,
		Message: message,
	}
}

func (e *APIError) Error() string {
	return e.Message
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
