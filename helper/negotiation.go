package helper

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func SuccessNegotiated(c *fiber.Ctx, status int, message string, data any) error {
	accept := strings.ToLower(c.Get("Accept"))

	if strings.Contains(accept, "text/csv") &&
		!strings.Contains(accept, "application/json") &&
		!strings.Contains(accept, "*/*") {
		return successCSV(c, status, data)
	}

	if accept != "" &&
		!strings.Contains(accept, "application/json") &&
		!strings.Contains(accept, "*/*") &&
		!strings.Contains(accept, "text/csv") {
		return Fail(c, fiber.StatusNotAcceptable, "format respons tidak didukung")
	}

	return Success(c, status, message, data)
}

func successCSV(c *fiber.Ctx, status int, data any) error {
	value := reflect.ValueOf(data)

	if !value.IsValid() {
		return Fail(c, fiber.StatusNotAcceptable, "data tidak dapat diubah ke CSV")
	}

	if value.Kind() == reflect.Map {
		items := value.MapIndex(reflect.ValueOf("items"))
		if !items.IsValid() {
			return Fail(c, fiber.StatusNotAcceptable, "data tidak dapat diubah ke CSV")
		}
		value = items
	}

	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return Fail(c, fiber.StatusNotAcceptable, "data tidak dapat diubah ke CSV")
		}
		value = value.Elem()
	}

	if value.Kind() != reflect.Slice {
		return Fail(c, fiber.StatusNotAcceptable, "data harus berupa daftar")
	}

	elemType := value.Type().Elem()
	for elemType.Kind() == reflect.Pointer || elemType.Kind() == reflect.Interface {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return Fail(c, fiber.StatusNotAcceptable, "data tidak dapat diubah ke CSV")
	}

	headers := []string{}
	fieldIndexes := []int{}

	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		if !field.IsExported() {
			continue
		}

		tag := strings.Split(field.Tag.Get("json"), ",")[0]
		if tag == "-" {
			continue
		}
		if tag == "" {
			tag = field.Name
		}

		headers = append(headers, tag)
		fieldIndexes = append(fieldIndexes, i)
	}

	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	if err := writer.Write(headers); err != nil {
		return Fail(c, fiber.StatusInternalServerError, "gagal membuat CSV")
	}

	for i := 0; i < value.Len(); i++ {
		item := value.Index(i)

		for item.Kind() == reflect.Pointer || item.Kind() == reflect.Interface {
			if item.IsNil() {
				return Fail(c, fiber.StatusInternalServerError, "data CSV tidak valid")
			}
			item = item.Elem()
		}

		record := make([]string, 0, len(fieldIndexes))

		for _, index := range fieldIndexes {
			field := item.Field(index)
			encoded, err := json.Marshal(field.Interface())
			if err != nil {
				return Fail(c, fiber.StatusInternalServerError, "gagal membuat CSV")
			}

			var cell any
			if err := json.Unmarshal(encoded, &cell); err != nil {
				return Fail(c, fiber.StatusInternalServerError, "gagal membuat CSV")
			}
			record = append(record, fmt.Sprint(cell))
		}

		if err := writer.Write(record); err != nil {
			return Fail(c, fiber.StatusInternalServerError, "gagal membuat CSV")
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return Fail(c, fiber.StatusInternalServerError, "gagal membuat CSV")
	}

	c.Set("Content-Type", "text/csv; charset=utf-8")
	return c.Status(status).Send(buffer.Bytes())
}
