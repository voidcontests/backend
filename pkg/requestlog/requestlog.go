package requestlog

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/cascadecontests/backend/internal/app/handler"
	"github.com/cascadecontests/backend/pkg/requestid"
	"github.com/labstack/echo/v4"
)

func Completed(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		slog.Info("request handler",
			slog.String("id", requestid.Get(c)),
			slog.String("method", c.Request().Method),
			slog.String("uri", c.Request().URL.Path),
			slog.String("client_ip", c.RealIP()),
			slog.String("host", c.Request().Host),
			slog.String("user_agent", c.Request().UserAgent()),
		)

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
