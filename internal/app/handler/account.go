package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	jwtgo "github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/voidcontests/api/internal/app/handler/dto/request"
	"github.com/voidcontests/api/internal/app/handler/dto/response"
	"github.com/voidcontests/api/internal/app/service"
	"github.com/voidcontests/api/internal/jwt"
	"github.com/voidcontests/api/internal/storage/models"
	"github.com/voidcontests/api/pkg/requestid"
	"github.com/voidcontests/api/pkg/validate"
)

func (h *Handler) CreateAccount(c echo.Context) error {
	ctx := c.Request().Context()

	var body request.CreateAccount
	if err := validate.Bind(c, &body); err != nil {
		return Error(http.StatusBadRequest, "invalid body: missing required fields")
	}

	id, err := h.service.Account.CreateAccount(ctx, body.Username, body.Password)
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			return Error(http.StatusConflict, "user already exists")
		}
		return err
	}

	return c.JSON(http.StatusCreated, response.ID{
		ID: id,
	})
}

func (h *Handler) CreateSession(c echo.Context) error {
	ctx := c.Request().Context()

	var body request.CreateSession
	if err := validate.Bind(c, &body); err != nil {
		return Error(http.StatusBadRequest, "invalid body: missing required fields")
	}

	token, err := h.service.Account.CreateSession(ctx, body.Username, body.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return Error(http.StatusUnauthorized, "invalid credentials")
		}
		return err
	}

	return c.JSON(http.StatusCreated, response.Token{
		Token: token,
	})
}

func (h *Handler) GetAccount(c echo.Context) error {
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	accountInfo, err := h.service.Account.GetAccount(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			return Error(http.StatusUnauthorized, "invalid or expired token")
		}
		return err
	}

	return c.JSON(http.StatusOK, response.Account{
		ID:       accountInfo.User.ID,
		Username: accountInfo.User.Username,
		Address:  accountInfo.User.Address,
		Role: response.Role{
			Name:                 accountInfo.Role.Name,
			CreatedProblemsLimit: accountInfo.Role.CreatedProblemsLimit,
			CreatedContestsLimit: accountInfo.Role.CreatedContestsLimit,
		},
	})
}

func (h *Handler) UpdateAccount(c echo.Context) error {
	ctx := c.Request().Context()

	claims, _ := ExtractClaims(c)

	var body request.UpdateAccount
	if err := validate.Bind(c, &body); err != nil {
		return Error(http.StatusBadRequest, "invalid body: missing required fields")
	}

	if body.Username == nil && body.Address == nil {
		return Error(http.StatusBadRequest, "at least one field must be provided")
	}

	params := models.UpdateUserParams{
		Username: body.Username,
		Address:  body.Address,
	}

	user, err := h.service.Account.UpdateAccount(ctx, claims.UserID, params)
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			return Error(http.StatusConflict, "username already taken")
		}
		if errors.Is(err, service.ErrInvalidToken) {
			return Error(http.StatusUnauthorized, "invalid or expired token")
		}
		return err
	}

	return c.JSON(http.StatusOK, response.User{
		ID:       user.ID,
		Username: user.Username,
		Address:  user.Address,
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
				if skiperr {
					return next(c)
				} else {
					return Error(http.StatusUnauthorized, "invalid or malformed token")
				}
			}

			if !token.Valid {
				if skiperr {
					return next(c)
				} else {
					return Error(http.StatusUnauthorized, "invalid or malformed token")
				}
			}

			claims, ok := token.Claims.(*jwt.CustomClaims)
			if !ok {
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
