package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/dto/request"
	"github.com/voidcontests/backend/internal/app/handler/dto/response"
	"github.com/voidcontests/backend/internal/app/service"
	"github.com/voidcontests/backend/pkg/validate"
)

func (h *Handler) CreateAccount(c echo.Context) error {
	ctx := c.Request().Context()

	var body request.CreateAccount
	if err := validate.Bind(c, &body); err != nil {
		return Error(http.StatusBadRequest, "invalid body: missing required fields")
	}

	id, err := h.accountService.CreateAccount(ctx, body)
	if errors.Is(err, service.ErrUserAlreadyExists) {
		return Error(http.StatusConflict, "user already exists")
	}
	if err != nil {
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

	token, err := h.accountService.CreateSession(ctx, body)
	if errors.Is(err, service.ErrUserNotFound) {
		return Error(http.StatusNotFound, "user not found")
	}
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, response.Token{
		Token: token,
	})
}

func (h *Handler) GetAccount(c echo.Context) error {
	ctx := c.Request().Context()
	claims, _ := ExtractClaims(c)

	user, err := h.accountService.GetAccount(ctx, claims.UserID)
	if errors.Is(err, service.ErrUserNotFound) {
		return Error(http.StatusNotFound, "user not found")
	}
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, response.Account{
		ID:       user.ID,
		Username: user.Username,
		Role: response.Role{
			Name:                 user.Role.Name,
			CreatedProblemsLimit: user.Role.CreatedProblemsLimit,
			CreatedContestsLimit: user.Role.CreatedContestsLimit,
		},
	})
}
