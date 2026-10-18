package helper

import (
	"reflect"
	"strings"

	playground "github.com/go-playground/validator/v10"
)

var validate = playground.New()

// ValidateStruct memvalidasi struct berdasarkan tag validate.
func ValidateStruct(data any) map[string]string {
	err := validate.Struct(data)
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(playground.ValidationErrors)
	if !ok {
		return map[string]string{
			"body": "data tidak valid",
		}
	}

	result := make(map[string]string)

	for _, fieldErr := range validationErrors {
		fieldName := getJSONFieldName(data, fieldErr.StructField())
		result[fieldName] = validationMessage(fieldErr)
	}

	return result
}

func getJSONFieldName(data any, structField string) string {
	value := reflect.ValueOf(data)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if value.IsValid() && value.Kind() == reflect.Struct {
		field, ok := value.Type().FieldByName(structField)
		if ok {
			tag := field.Tag.Get("json")
			name := strings.Split(tag, ",")[0]
			if name != "" && name != "-" {
				return name
			}
		}
	}

	return strings.ToLower(structField)
}

func validationMessage(fieldErr playground.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return "field wajib diisi"
	case "email":
		return "format email tidak valid"
	case "min":
		return "minimal " + fieldErr.Param() + " karakter"
	case "max":
		return "maksimal " + fieldErr.Param() + " karakter"
	default:
		return "format field tidak valid"
	}
}
