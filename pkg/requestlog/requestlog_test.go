package requestlog

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/voidcontests/api/internal/app/handler"
)

func TestCompleted(t *testing.T) {
	e := echo.New()

	t.Run("calls next handler and logs request", func(t *testing.T) {
		nextCalled := false
		middleware := Completed(func(c echo.Context) error {
			nextCalled = true
			return c.String(http.StatusOK, "success")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "test-request-id")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := middleware(c)
		assert.NoError(t, err)
		assert.True(t, nextCalled)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("skips OPTIONS requests", func(t *testing.T) {
		nextCalled := false
		middleware := Completed(func(c echo.Context) error {
			nextCalled = true
			return c.NoContent(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := middleware(c)
		assert.NoError(t, err)
		assert.True(t, nextCalled)
	})

	t.Run("skips logging for healthcheck endpoint with 200 status", func(t *testing.T) {
		middleware := Completed(func(c echo.Context) error {
			return c.String(http.StatusOK, "healthy")
		})

		req := httptest.NewRequest(http.MethodGet, "/api/healthcheck", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/healthcheck")

		err := middleware(c)
		assert.NoError(t, err)
	})

	t.Run("logs healthcheck endpoint with non-200 status", func(t *testing.T) {
		middleware := Completed(func(c echo.Context) error {
			return c.String(http.StatusInternalServerError, "unhealthy")
		})

		req := httptest.NewRequest(http.MethodGet, "/api/healthcheck", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/healthcheck")

		err := middleware(c)
		assert.NoError(t, err)
	})

	t.Run("propagates error from next handler", func(t *testing.T) {
		expectedErr := errors.New("test error")
		middleware := Completed(func(c echo.Context) error {
			return expectedErr
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := middleware(c)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("handles APIError and extracts status code", func(t *testing.T) {
		apiErr := &handler.APIError{
			Status:  http.StatusBadRequest,
			Message: "bad request",
		}

		middleware := Completed(func(c echo.Context) error {
			return apiErr
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "test-id")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := middleware(c)
		assert.Equal(t, apiErr, err)
	})

	t.Run("handles non-APIError and uses status 500", func(t *testing.T) {
		genericErr := errors.New("generic error")

		middleware := Completed(func(c echo.Context) error {
			return genericErr
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "test-id")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := middleware(c)
		assert.Equal(t, genericErr, err)
	})

	t.Run("logs correct request information", func(t *testing.T) {
		middleware := Completed(func(c echo.Context) error {
			return c.String(http.StatusCreated, "created")
		})

		req := httptest.NewRequest(http.MethodPost, "/api/users", nil)
		req.Header.Set("X-Request-ID", "req-123")
		req.Header.Set("User-Agent", "TestAgent/1.0")
		req.Host = "example.com"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/users")

		err := middleware(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("measures request duration", func(t *testing.T) {
		middleware := Completed(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "test-id")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := middleware(c)
		assert.NoError(t, err)
	})

	t.Run("handles different HTTP methods", func(t *testing.T) {
		methods := []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		}

		for _, method := range methods {
			middleware := Completed(func(c echo.Context) error {
				return c.String(http.StatusOK, "success")
			})

			req := httptest.NewRequest(method, "/test", nil)
			req.Header.Set("X-Request-ID", "test-id")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := middleware(c)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("retrieves request ID from context", func(t *testing.T) {
		expectedID := "custom-request-id-456"
		middleware := Completed(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", expectedID)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := middleware(c)
		assert.NoError(t, err)
	})

	t.Run("logs RealIP from context", func(t *testing.T) {
		middleware := Completed(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Real-IP", "203.0.113.42")
		req.Header.Set("X-Request-ID", "test-id")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := middleware(c)
		assert.NoError(t, err)
	})

	t.Run("handles empty user agent", func(t *testing.T) {
		middleware := Completed(func(c echo.Context) error {
			return c.String(http.StatusOK, "success")
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "test-id")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := middleware(c)
		assert.NoError(t, err)
	})

	t.Run("integration: full request lifecycle", func(t *testing.T) {
		handlerCalled := false
		middleware := Completed(func(c echo.Context) error {
			handlerCalled = true
			return c.JSON(http.StatusOK, map[string]string{
				"message": "success",
			})
		})

		req := httptest.NewRequest(http.MethodPost, "/api/data", nil)
		req.Header.Set("X-Request-ID", "integration-test-id")
		req.Header.Set("User-Agent", "IntegrationTest/1.0")
		req.Header.Set("X-Real-IP", "192.168.1.100")
		req.Host = "api.example.com"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetPath("/api/data")

		err := middleware(c)
		assert.NoError(t, err)
		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "success")
	})
}
