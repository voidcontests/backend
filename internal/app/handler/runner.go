package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/app/handler/dto/request"
	"github.com/voidcontests/backend/internal/lib/logger/sl"
	"github.com/voidcontests/backend/internal/runner"
	"github.com/voidcontests/backend/pkg/requestid"
	"github.com/voidcontests/backend/pkg/validate"
)

func (h *Handler) ExecuteSolution(c echo.Context) error {
	requestID := requestid.Get(c)
	log := slog.With(slog.String("op", "handler.ExecuteSolution"), slog.String("request_id", requestID))

	var body request.CreateCodeSubmission
	if err := validate.Bind(c, &body); err != nil {
		log.Debug("can't decode request body", sl.Err(err))
		return Error(http.StatusBadRequest, "invalid body: missing required fields")
	}

	// TODO: Use submission.ID instead of request id
	filename := fmt.Sprintf("%s.c", requestID)

	file, err := os.Create(filename)
	if err != nil {
		log.Error("failed to create file: %w", sl.Err(err))
		return err
	}
	defer file.Close()

	_, err = file.WriteString(body.Code)
	if err != nil {
		log.Error("failed to write to file: %w", sl.Err(err))
		return err
	}

	res, err := runner.Execute(filename)
	if err != nil {
		return err
	}

	err = os.Remove(filename)
	if err != nil {
		log.Error("couldn't delete source file", sl.Err(err))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"exit_code": res.ExitCode,
		"stdout":    res.Stdout,
		"stderr":    res.Stderr,
	})
}
