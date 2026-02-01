package requestid

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	e := echo.New()

	t.Run("generates and sets request ID when not present", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		called := false
		handler := New(func(c echo.Context) error {
			called = true
			rid := c.Request().Header.Get(headerRequestID)
			assert.NotEmpty(t, rid)
			_, err := uuid.Parse(rid)
			assert.NoError(t, err)
			return nil
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("preserves existing request ID from response header", func(t *testing.T) {
		existingID := "existing-request-id"
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		rec.Header().Set(headerRequestID, existingID)
		c := e.NewContext(req, rec)

		called := false
		handler := New(func(c echo.Context) error {
			called = true
			return nil
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.True(t, called)
	})

	t.Run("calls next handler", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		nextCalled := false
		handler := New(func(c echo.Context) error {
			nextCalled = true
			return nil
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.True(t, nextCalled)
	})

	t.Run("propagates error from next handler", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		expectedErr := echo.NewHTTPError(http.StatusInternalServerError, "test error")
		handler := New(func(c echo.Context) error {
			return expectedErr
		})

		err := handler(c)
		assert.Equal(t, expectedErr, err)
	})
}

func TestGet(t *testing.T) {
	e := echo.New()

	t.Run("returns request ID from header", func(t *testing.T) {
		expectedID := "test-request-id-123"
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(headerRequestID, expectedID)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		rid := Get(c)
		assert.Equal(t, expectedID, rid)
	})

	t.Run("returns empty string when request ID not set", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		rid := Get(c)
		assert.Equal(t, "", rid)
	})

	t.Run("integration: Get returns ID set by New middleware", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		var capturedID string
		handler := New(func(c echo.Context) error {
			capturedID = Get(c)
			return nil
		})

		err := handler(c)
		assert.NoError(t, err)
		assert.NotEmpty(t, capturedID)

		_, err = uuid.Parse(capturedID)
		assert.NoError(t, err)
	})
}
