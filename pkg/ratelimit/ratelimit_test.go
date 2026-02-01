package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestWithTimeout(t *testing.T) {
	e := echo.New()

	t.Run("allows first request from IP", func(t *testing.T) {
		middleware := WithTimeout(1 * time.Second)
		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Real-IP", "192.168.1.1")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := handler(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("blocks second request from same IP within timeout", func(t *testing.T) {
		middleware := WithTimeout(500 * time.Millisecond)
		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		req1 := httptest.NewRequest(http.MethodGet, "/", nil)
		req1.Header.Set("X-Real-IP", "192.168.1.2")
		rec1 := httptest.NewRecorder()
		c1 := e.NewContext(req1, rec1)

		err := handler(c1)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec1.Code)

		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.Header.Set("X-Real-IP", "192.168.1.2")
		rec2 := httptest.NewRecorder()
		c2 := e.NewContext(req2, rec2)

		err = handler(c2)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusTooManyRequests, rec2.Code)
	})

	t.Run("allows request after timeout expires", func(t *testing.T) {
		duration := 100 * time.Millisecond
		middleware := WithTimeout(duration)
		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		req1 := httptest.NewRequest(http.MethodGet, "/", nil)
		req1.Header.Set("X-Real-IP", "192.168.1.3")
		rec1 := httptest.NewRecorder()
		c1 := e.NewContext(req1, rec1)

		err := handler(c1)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec1.Code)

		time.Sleep(duration + 10*time.Millisecond)

		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.Header.Set("X-Real-IP", "192.168.1.3")
		rec2 := httptest.NewRecorder()
		c2 := e.NewContext(req2, rec2)

		err = handler(c2)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec2.Code)
	})

	t.Run("tracks different IPs independently", func(t *testing.T) {
		middleware := WithTimeout(1 * time.Second)
		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		req1 := httptest.NewRequest(http.MethodGet, "/", nil)
		req1.Header.Set("X-Real-IP", "192.168.1.4")
		rec1 := httptest.NewRecorder()
		c1 := e.NewContext(req1, rec1)

		err := handler(c1)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec1.Code)

		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.Header.Set("X-Real-IP", "192.168.1.5")
		rec2 := httptest.NewRecorder()
		c2 := e.NewContext(req2, rec2)

		err = handler(c2)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec2.Code)
	})

	t.Run("returns JSON error response with timeout information", func(t *testing.T) {
		duration := 2 * time.Second
		middleware := WithTimeout(duration)
		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		req1 := httptest.NewRequest(http.MethodGet, "/", nil)
		req1.Header.Set("X-Real-IP", "192.168.1.6")
		rec1 := httptest.NewRecorder()
		c1 := e.NewContext(req1, rec1)

		_ = handler(c1)

		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.Header.Set("X-Real-IP", "192.168.1.6")
		rec2 := httptest.NewRecorder()
		c2 := e.NewContext(req2, rec2)

		err := handler(c2)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusTooManyRequests, rec2.Code)
		assert.Contains(t, rec2.Body.String(), "rate limit exceeded")
		assert.Contains(t, rec2.Body.String(), "timeout")
	})

	t.Run("uses RealIP from context", func(t *testing.T) {
		middleware := WithTimeout(500 * time.Millisecond)
		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		req1 := httptest.NewRequest(http.MethodGet, "/", nil)
		req1.Header.Set("X-Forwarded-For", "10.0.0.1")
		rec1 := httptest.NewRecorder()
		c1 := e.NewContext(req1, rec1)

		err := handler(c1)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec1.Code)

		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.Header.Set("X-Forwarded-For", "10.0.0.1")
		rec2 := httptest.NewRecorder()
		c2 := e.NewContext(req2, rec2)

		err = handler(c2)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusTooManyRequests, rec2.Code)
	})

	t.Run("allows multiple requests within sequence correctly", func(t *testing.T) {
		duration := 100 * time.Millisecond
		middleware := WithTimeout(duration)
		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		ip := "192.168.1.7"

		req1 := httptest.NewRequest(http.MethodGet, "/", nil)
		req1.Header.Set("X-Real-IP", ip)
		rec1 := httptest.NewRecorder()
		c1 := e.NewContext(req1, rec1)
		err := handler(c1)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec1.Code)

		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.Header.Set("X-Real-IP", ip)
		rec2 := httptest.NewRecorder()
		c2 := e.NewContext(req2, rec2)
		err = handler(c2)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusTooManyRequests, rec2.Code)

		time.Sleep(duration + 10*time.Millisecond)

		req3 := httptest.NewRequest(http.MethodGet, "/", nil)
		req3.Header.Set("X-Real-IP", ip)
		rec3 := httptest.NewRecorder()
		c3 := e.NewContext(req3, rec3)
		err = handler(c3)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec3.Code)
	})

	t.Run("handles concurrent requests from different IPs", func(t *testing.T) {
		middleware := WithTimeout(1 * time.Second)
		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		successCount := 0
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(index int) {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("X-Real-IP", "192.168.1."+string(rune(100+index)))
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				_ = handler(c)
				if rec.Code == http.StatusOK {
					done <- true
				} else {
					done <- false
				}
			}(i)
		}

		for i := 0; i < 10; i++ {
			if <-done {
				successCount++
			}
		}

		assert.Equal(t, 10, successCount)
	})

	t.Run("timeout value in response is accurate", func(t *testing.T) {
		duration := 5 * time.Second
		middleware := WithTimeout(duration)
		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		req1 := httptest.NewRequest(http.MethodGet, "/", nil)
		req1.Header.Set("X-Real-IP", "192.168.1.8")
		rec1 := httptest.NewRecorder()
		c1 := e.NewContext(req1, rec1)
		_ = handler(c1)

		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.Header.Set("X-Real-IP", "192.168.1.8")
		rec2 := httptest.NewRecorder()
		c2 := e.NewContext(req2, rec2)
		_ = handler(c2)

		body := rec2.Body.String()
		assert.Contains(t, body, "timeout")
		assert.Contains(t, body, "5s")
	})
}
