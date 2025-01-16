package validate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/labstack/echo/v4"
)

func Bind(c echo.Context, dst interface{}) error {
	if reflect.TypeOf(dst).Kind() != reflect.Ptr {
		return errors.New("validate: invalid dstination: expected pointer")
	}

	b, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return fmt.Errorf("io.ReadAll: %v", err)
	}

	json.Unmarshal(b, dst)
	if err := c.Bind(dst); err != nil {
		return fmt.Errorf("json.Unmarshall: %v", err)
	}

	return Struct(dst)
}

func Struct(dst interface{}) error {
	val := reflect.ValueOf(dst)
	typ := reflect.TypeOf(dst)

	for i := 0; i < val.Elem().NumField(); i++ {
		field := val.Elem().Field(i)
		fieldType := typ.Elem().Field(i)

		if fieldType.Tag.Get("required") == "true" {
			name := fieldType.Tag.Get("json")
			if isEmptyValue(field) {
				return fmt.Errorf("`%s` is required", name)
			}
		}
	}
	return nil
}

func isEmptyValue(v reflect.Value) bool {
	return v.String() == "" || (v.Kind() == reflect.Int && v.Int() <= 0)
}
