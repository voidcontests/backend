package validate

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/labstack/echo/v4"
)

func Bind(c echo.Context, dst interface{}) error {
	if reflect.TypeOf(dst).Kind() != reflect.Ptr {
		return errors.New("validate: invalid dstination: expected pointer")
	}

	// NOTE: Validate used only on parsing JSON data, so it makes sense to set an `application/json` header here.
	// And sometimes I just don't want to set it manualy in curl
	c.Request().Header.Set("Content-Type", "application/json")
	if err := c.Bind(dst); err != nil {
		return fmt.Errorf("json.Unmarshall: %v", err)
	}

	return Struct(dst)
}

func Struct(dst interface{}) error {
	val := reflect.ValueOf(dst)
	typ := reflect.TypeOf(dst)

	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("expected a pointer to a struct")
	}

	val = val.Elem()
	typ = typ.Elem()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if fieldType.Tag.Get("required") == "true" {
			name := fieldType.Tag.Get("json")
			if isEmptyValue(field) {
				return fmt.Errorf("`%s` is required", name)
			}
		}

		if field.Kind() == reflect.Struct && fieldType.Type != reflect.TypeOf(time.Time{}) {
			var nestedPtr interface{}
			if field.CanAddr() {
				nestedPtr = field.Addr().Interface()
			} else {
				nestedCopy := reflect.New(field.Type()).Elem()
				nestedCopy.Set(field)
				nestedPtr = nestedCopy.Addr().Interface()
			}

			if err := Struct(nestedPtr); err != nil {
				return fmt.Errorf("error in nested struct `%s`: %w", fieldType.Name, err)
			}
		}
	}

	return nil
}

func isEmptyValue(v reflect.Value) bool {
	return v.String() == "" || (v.Kind() == reflect.Int && v.Int() <= 0)
}
