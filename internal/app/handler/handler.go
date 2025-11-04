package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/api/internal/app/service"
	"github.com/voidcontests/api/internal/config"
	"github.com/voidcontests/api/internal/jwt"
	"github.com/voidcontests/api/internal/storage/broker"
	"github.com/voidcontests/api/internal/storage/repository"
	"github.com/voidcontests/api/pkg/ton"
)

type Handler struct {
	config  *config.Config
	repo    *repository.Repository
	broker  broker.Broker
	service *service.Service
}

func New(c *config.Config, r *repository.Repository, b broker.Broker, tc *ton.Client) *Handler {
	return &Handler{
		config:  c,
		repo:    r,
		broker:  b,
		service: service.New(&c.Security, r, b, tc),
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
