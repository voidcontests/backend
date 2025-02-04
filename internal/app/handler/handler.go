package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/tonkeeper/tongo/tonconnect"
	"github.com/voidcontests/backend/internal/config"
	"github.com/voidcontests/backend/internal/jwt"
	"github.com/voidcontests/backend/internal/repository"
)

type Handler struct {
	config            *config.Config
	repo              *repository.Repository
	tonconnectMainnet *tonconnect.Server
	tonconnectTestnet *tonconnect.Server
}

func New(c *config.Config, r *repository.Repository, mainnet, testnet *tonconnect.Server) *Handler {
	return &Handler{
		config:            c,
		repo:              r,
		tonconnectMainnet: mainnet,
		tonconnectTestnet: testnet,
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
