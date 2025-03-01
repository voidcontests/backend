package handler

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/runner"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/pkg/requestid"
	"github.com/voidcontests/backend/pkg/validate"
)

func (h *Handler) Run(c echo.Context) error {
	log := slog.With(slog.String("op", "handler.Run"), slog.String("request_id", requestid.Get(c)))

	var body struct {
		Code  string `json:"code" required:"true"`
		Input string `json:"input"`
	}
	if err := validate.Bind(c, &body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		return Error(http.StatusBadRequest, "invalid body: missing required fields")
	}

	result, err := runner.ExecuteWithInput(body.Code, body.Input)
	if err != nil {
		log.Error("can't execute solution", sl.Err(err))
	}

	return c.JSON(http.StatusOK, result)
}
