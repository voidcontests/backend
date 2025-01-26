package ratelimit

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// WithTimeout returns a middleware that allows only one request per `duration` from the same IP.
func WithTimeout(duration time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		lastRequests := make(map[string]time.Time)

		return func(c echo.Context) error {
			ip := c.RealIP()
			if lastRequest, ok := lastRequests[ip]; ok {
				since := time.Since(lastRequest)
				if since < duration {
					slog.Debug("rate limited", slog.String("ip", ip))
					return c.JSON(http.StatusTooManyRequests, map[string]any{
						"timeout": fmt.Sprintf("%ds", int64((duration - since).Seconds())),
					})
				}
			}

			lastRequests[ip] = time.Now()

			return next(c)
		}
	}
}
