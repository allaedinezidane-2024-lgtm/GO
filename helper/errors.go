package helper

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"
)

type AppError struct {
	Status  int
	Code    string
	Message string
	Errors  any
}

func (e *AppError) Error() string {
	return e.Message
}

type ErrorResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Code      string `json:"code"`
	RequestID string `json:"request_id"`
	Errors    any    `json:"errors,omitempty"`
}

func errorCode(status int) string {
	switch status {
	case fiber.StatusBadRequest:
		return "BAD_REQUEST"
	case fiber.StatusUnauthorized:
		return "UNAUTHORIZED"
	case fiber.StatusForbidden:
		return "FORBIDDEN"
	case fiber.StatusNotFound:
		return "NOT_FOUND"
	case fiber.StatusConflict:
		return "CONFLICT"
	case fiber.StatusUnsupportedMediaType:
		return "UNSUPPORTED_MEDIA_TYPE"
	case fiber.StatusNotAcceptable:
		return "NOT_ACCEPTABLE"
	case fiber.StatusUnprocessableEntity:
		return "VALIDATION_ERROR"
	default:
		return "INTERNAL_SERVER_ERROR"
	}
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	message := "Terjadi kesalahan pada server"
	code := "INTERNAL_SERVER_ERROR"
	var validationErrors any

	var appErr *AppError
	if errors.As(err, &appErr) {
		status = appErr.Status
		message = appErr.Message
		code = appErr.Code
		validationErrors = appErr.Errors
	} else {
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			status = fiberErr.Code
			message = fiberErr.Message
			code = errorCode(status)
		}
	}

	if status >= 500 {
		log.Printf(
			"request_id=%v method=%s path=%s status=%d error=%v",
			c.Locals("requestid"),
			c.Method(),
			c.Path(),
			status,
			err,
		)
	}

	requestID := c.Get("X-Request-ID")
	if id, ok := c.Locals("requestid").(string); ok && id != "" {
		requestID = id
	}

	return c.Status(status).JSON(ErrorResponse{
		Success:   false,
		Message:   message,
		Code:      code,
		RequestID: requestID,
		Errors:    validationErrors,
	})
}
