package validate

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestBind(t *testing.T) {
	e := echo.New()

	t.Run("successfully binds and validates valid JSON", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name" required:"true"`
			Email string `json:"email" required:"true"`
			Age   int    `json:"age"`
		}

		jsonBody := `{"name":"John","email":"john@example.com","age":30}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(jsonBody))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		var result TestStruct
		err := Bind(c, &result)

		assert.NoError(t, err)
		assert.Equal(t, "John", result.Name)
		assert.Equal(t, "john@example.com", result.Email)
		assert.Equal(t, 30, result.Age)
	})

	t.Run("returns error when required field is missing", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name" required:"true"`
			Email string `json:"email" required:"true"`
		}

		jsonBody := `{"name":"John"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(jsonBody))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		var result TestStruct
		err := Bind(c, &result)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email")
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("returns error when destination is not a pointer", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
		}

		jsonBody := `{"name":"John"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(jsonBody))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		var result TestStruct
		err := Bind(c, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pointer")
	})

	t.Run("returns error on invalid JSON", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
		}

		jsonBody := `{"name":invalid}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(jsonBody))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		var result TestStruct
		err := Bind(c, &result)

		assert.Error(t, err)
	})

	t.Run("sets Content-Type header to application/json", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
		}

		jsonBody := `{"name":"John"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(jsonBody))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		var result TestStruct
		_ = Bind(c, &result)

		assert.Equal(t, "application/json", c.Request().Header.Get("Content-Type"))
	})
}

func TestStruct(t *testing.T) {
	t.Run("validates struct with all required fields present", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name" required:"true"`
			Email string `json:"email" required:"true"`
			Age   int    `json:"age" required:"true"`
		}

		data := &TestStruct{
			Name:  "John",
			Email: "john@example.com",
			Age:   30,
		}

		err := Struct(data)
		assert.NoError(t, err)
	})

	t.Run("returns error when string field is empty", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name" required:"true"`
		}

		data := &TestStruct{Name: ""}
		err := Struct(data)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name")
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("returns error when int field is zero or negative", func(t *testing.T) {
		type TestStruct struct {
			Age int `json:"age" required:"true"`
		}

		data := &TestStruct{Age: 0}
		err := Struct(data)
		assert.Error(t, err)

		data = &TestStruct{Age: -1}
		err = Struct(data)
		assert.Error(t, err)
	})

	t.Run("validates positive int field", func(t *testing.T) {
		type TestStruct struct {
			Age int `json:"age" required:"true"`
		}

		data := &TestStruct{Age: 30}
		err := Struct(data)
		assert.NoError(t, err)
	})

	t.Run("returns error when slice is empty", func(t *testing.T) {
		type TestStruct struct {
			Tags []string `json:"tags" required:"true"`
		}

		data := &TestStruct{Tags: []string{}}
		err := Struct(data)
		assert.Error(t, err)
	})

	t.Run("validates non-empty slice", func(t *testing.T) {
		type TestStruct struct {
			Tags []string `json:"tags" required:"true"`
		}

		data := &TestStruct{Tags: []string{"tag1", "tag2"}}
		err := Struct(data)
		assert.NoError(t, err)
	})

	t.Run("validates nested structs", func(t *testing.T) {
		type Address struct {
			Street string `json:"street" required:"true"`
			City   string `json:"city" required:"true"`
		}

		type Person struct {
			Name    string  `json:"name" required:"true"`
			Address Address `json:"address"`
		}

		data := &Person{
			Name: "John",
			Address: Address{
				Street: "123 Main St",
				City:   "New York",
			},
		}

		err := Struct(data)
		assert.NoError(t, err)
	})

	t.Run("returns error when nested struct has missing required field", func(t *testing.T) {
		type Address struct {
			Street string `json:"street" required:"true"`
			City   string `json:"city" required:"true"`
		}

		type Person struct {
			Name    string  `json:"name" required:"true"`
			Address Address `json:"address"`
		}

		data := &Person{
			Name: "John",
			Address: Address{
				Street: "123 Main St",
				City:   ""},
		}

		err := Struct(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Address")
		assert.Contains(t, err.Error(), "city")
	})

	t.Run("handles time.Time fields without validating them as structs", func(t *testing.T) {
		type TestStruct struct {
			Name      string    `json:"name" required:"true"`
			CreatedAt time.Time `json:"created_at"`
		}

		data := &TestStruct{
			Name:      "John",
			CreatedAt: time.Now(),
		}

		err := Struct(data)
		assert.NoError(t, err)
	})

	t.Run("returns error when destination is not a pointer", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
		}

		data := TestStruct{Name: "John"}
		err := Struct(data)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pointer")
	})

	t.Run("returns error when destination is not a struct", func(t *testing.T) {
		data := "not a struct"
		err := Struct(&data)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "struct")
	})

	t.Run("validates pointer fields", func(t *testing.T) {
		type TestStruct struct {
			Name *string `json:"name" required:"true"`
		}

		name := "John"
		data := &TestStruct{Name: &name}
		err := Struct(data)
		assert.NoError(t, err)
	})

	t.Run("returns error when pointer field is nil", func(t *testing.T) {
		type TestStruct struct {
			Name *string `json:"name" required:"true"`
		}

		data := &TestStruct{Name: nil}
		err := Struct(data)
		assert.Error(t, err)
	})

	t.Run("validates map fields", func(t *testing.T) {
		type TestStruct struct {
			Metadata map[string]string `json:"metadata" required:"true"`
		}

		data := &TestStruct{Metadata: map[string]string{"key": "value"}}
		err := Struct(data)
		assert.NoError(t, err)
	})

	t.Run("returns error when map field is empty", func(t *testing.T) {
		type TestStruct struct {
			Metadata map[string]string `json:"metadata" required:"true"`
		}

		data := &TestStruct{Metadata: map[string]string{}}
		err := Struct(data)
		assert.Error(t, err)
	})

	t.Run("validates float fields", func(t *testing.T) {
		type TestStruct struct {
			Price float64 `json:"price" required:"true"`
		}

		data := &TestStruct{Price: 9.99}
		err := Struct(data)
		assert.NoError(t, err)
	})

	t.Run("returns error when float field is zero or negative", func(t *testing.T) {
		type TestStruct struct {
			Price float64 `json:"price" required:"true"`
		}

		data := &TestStruct{Price: 0.0}
		err := Struct(data)
		assert.Error(t, err)

		data = &TestStruct{Price: -1.5}
		err = Struct(data)
		assert.Error(t, err)
	})

	t.Run("validates uint fields", func(t *testing.T) {
		type TestStruct struct {
			Count uint `json:"count" required:"true"`
		}

		data := &TestStruct{Count: 10}
		err := Struct(data)
		assert.NoError(t, err)
	})

	t.Run("returns error when uint field is zero", func(t *testing.T) {
		type TestStruct struct {
			Count uint `json:"count" required:"true"`
		}

		data := &TestStruct{Count: 0}
		err := Struct(data)
		assert.Error(t, err)
	})

	t.Run("ignores fields without required tag", func(t *testing.T) {
		type TestStruct struct {
			Name     string `json:"name" required:"true"`
			Optional string `json:"optional"`
		}

		data := &TestStruct{
			Name:     "John",
			Optional: ""}

		err := Struct(data)
		assert.NoError(t, err)
	})
}
