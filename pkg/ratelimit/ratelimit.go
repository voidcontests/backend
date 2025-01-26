package ratelimit

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

// WithTimeout returns a middleware that allows only one request per `duration` from the same IP.
func WithTimeout(duration time.Duration) echo.MiddlewareFunc {
	lastRequests := make(map[string]time.Time)
	var mu sync.Mutex

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()

			mu.Lock()
			defer mu.Unlock()

			if lastRequest, ok := lastRequests[ip]; ok {
				since := time.Since(lastRequest)
				if since < duration {
					slog.Debug("rate limited", slog.String("ip", ip))
					return c.JSON(http.StatusTooManyRequests, map[string]any{
						"timeout": fmt.Sprintf("%ds", int64(duration.Seconds()-since.Seconds())),
					})
				}
			}

			lastRequests[ip] = time.Now()

			return next(c)
		}
	}
}
