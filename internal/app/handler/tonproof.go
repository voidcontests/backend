package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/api/internal/app/handler/dto/request"
	"github.com/voidcontests/api/pkg/validate"
)

func (h *Handler) GeneratePayload(c echo.Context) error {
	payload, err := h.service.TonProof.GeneratePayload()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]any{
		"payload": payload,
	})
}

func (h *Handler) CheckProof(c echo.Context) error {
	ctx := c.Request().Context()

	claims, ok := ExtractClaims(c)
	if !ok {
		return Error(http.StatusUnauthorized, "user not authenticated")
	}

	var tp request.TonProof
	if err := validate.Bind(c, &tp); err != nil {
		return Error(http.StatusBadRequest, "invalid request body")
	}

	err := h.service.TonProof.VerifyProofAndSetAddress(ctx, claims.UserID, tp)
	if err != nil {
		return Error(http.StatusUnauthorized, "tonproof verification failed")
	}

	return c.NoContent(http.StatusOK)
}
