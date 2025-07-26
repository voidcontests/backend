package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/voidcontests/backend/internal/pkg/metrics"
)

func Metrics() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Path() == "/api/metrics" {
				return next(c)
			}

			start := time.Now()
			err := next(c)
			status := c.Response().Status
			duration := time.Since(start).Seconds()

			metrics.HttpRequestsTotal.WithLabelValues(
				c.Request().Method,
				c.Path(),
				httpStatusCodeToText(status),
			).Inc()

			metrics.RequestDuration.WithLabelValues(
				c.Request().Method,
				c.Path(),
			).Observe(duration)

			return err
		}
	}
}

func httpStatusCodeToText(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}
