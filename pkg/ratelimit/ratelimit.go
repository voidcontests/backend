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

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			mu.Lock()
			now := time.Now()
			for ip, lastRequest := range lastRequests {
				if now.Sub(lastRequest) > duration*2 {
					delete(lastRequests, ip)
				}
			}
			mu.Unlock()
		}
	}()

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
						"message": "rate limit exceeded",
						"timeout": fmt.Sprintf("%ds", int64(duration.Seconds()-since.Seconds())+1),
					})
				}
			}

			lastRequests[ip] = time.Now()

			return next(c)
		}
	}
}
