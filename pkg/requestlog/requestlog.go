package requestlog

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/api/internal/app/handler"
	"github.com/voidcontests/api/pkg/requestid"
)

func Completed(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Request().Method == "OPTIONS" {
			return next(c)
		}

		start := time.Now()
		err := next(c)

		status := c.Response().Status
		if err != nil {
			if apiErr, ok := err.(*handler.APIError); ok {
				status = apiErr.Status
			} else {
				status = 500
			}
		}

		if c.Path() == "/api/healthcheck" && status == 200 {
			return err
		}

		slog.Info("request completed",
			slog.String("id", requestid.Get(c)),
			slog.String("method", c.Request().Method),
			slog.String("uri", c.Request().URL.Path),
			slog.String("client_ip", c.RealIP()),
			slog.String("duration", fmt.Sprintf("%v", time.Since(start))),
			slog.String("host", c.Request().Host),
			slog.String("user_agent", c.Request().UserAgent()),
			slog.Int("status", status),
		)

		return err
	}
}
