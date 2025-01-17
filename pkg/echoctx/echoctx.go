package echoctx

import "github.com/labstack/echo/v4"

func Lookup[T any](c echo.Context, key string) (T, bool) {
	var zero T
	v := c.Get(key)
	if v == nil {
		return zero, false
	}

	t, ok := v.(T)
	if !ok {
		return zero, false
	}

	return t, true
}
