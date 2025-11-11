package echoctx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestLookup(t *testing.T) {
	e := echo.New()

	t.Run("returns value and true when key exists with correct type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		c.Set("test-string", "hello")
		c.Set("test-int", 42)
		c.Set("test-bool", true)

		str, ok := Lookup[string](c, "test-string")
		assert.True(t, ok)
		assert.Equal(t, "hello", str)

		num, ok := Lookup[int](c, "test-int")
		assert.True(t, ok)
		assert.Equal(t, 42, num)

		boolVal, ok := Lookup[bool](c, "test-bool")
		assert.True(t, ok)
		assert.True(t, boolVal)
	})

	t.Run("returns zero value and false when key does not exist", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		str, ok := Lookup[string](c, "nonexistent")
		assert.False(t, ok)
		assert.Equal(t, "", str)

		num, ok := Lookup[int](c, "nonexistent")
		assert.False(t, ok)
		assert.Equal(t, 0, num)
	})

	t.Run("returns zero value and false when type mismatch", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		c.Set("test-string", "hello")

		// Try to get string value as int
		num, ok := Lookup[int](c, "test-string")
		assert.False(t, ok)
		assert.Equal(t, 0, num)

		// Try to get string value as bool
		boolVal, ok := Lookup[bool](c, "test-string")
		assert.False(t, ok)
		assert.False(t, boolVal)
	})

	t.Run("works with custom types", func(t *testing.T) {
		type CustomStruct struct {
			Name string
			Age  int
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		expected := CustomStruct{Name: "John", Age: 30}
		c.Set("custom", expected)

		result, ok := Lookup[CustomStruct](c, "custom")
		assert.True(t, ok)
		assert.Equal(t, expected, result)
	})

	t.Run("works with pointer types", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		str := "hello"
		c.Set("ptr", &str)

		ptr, ok := Lookup[*string](c, "ptr")
		assert.True(t, ok)
		assert.NotNil(t, ptr)
		assert.Equal(t, "hello", *ptr)
	})

	t.Run("returns false when value is nil", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		c.Set("nil-value", nil)

		str, ok := Lookup[string](c, "nil-value")
		assert.False(t, ok)
		assert.Equal(t, "", str)
	})
}
