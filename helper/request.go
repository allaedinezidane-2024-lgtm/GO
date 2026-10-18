package helper

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model"
)

type cursorPayload struct {
	CreatedAt time.Time `json:"created_at"`
	ID        int       `json:"id"`
}

func EncodeCursor(createdAt time.Time, id int) (string, error) {
	payload, err := json.Marshal(cursorPayload{
		CreatedAt: createdAt,
		ID:        id,
	})
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func DecodeCursor(raw string) (*time.Time, int, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, 0, err
	}

	var cursor cursorPayload
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return nil, 0, err
	}
	if cursor.ID < 1 || cursor.CreatedAt.IsZero() {
		return nil, 0, fiber.ErrBadRequest
	}

	return &cursor.CreatedAt, cursor.ID, nil
}

func RequestContext(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 5*time.Second)
}

func ParamID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

var allowedSort = map[string]bool{
	"id":         true,
	"username":   true,
	"email":      true,
	"created_at": true,
}

func ParseListQuery(c *fiber.Ctx) model.ListQuery {
	q := model.ListQuery{
		Page:   c.QueryInt("page", 1),
		Limit:  c.QueryInt("limit", 10),
		Search: strings.TrimSpace(c.Query("search")),
		Sort:   c.Query("sort", "id"),
		Order:  strings.ToLower(c.Query("order", "asc")),
	}

	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 {
		q.Limit = 10
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	if !allowedSort[q.Sort] {
		q.Sort = "id"
	}
	if q.Order != "desc" {
		q.Order = "asc"
	}

	if raw := c.Query("is_active"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			q.IsActive = &v
		}
	}

	if raw := c.Query("cursor"); raw != "" {
		cursorTime, cursorID, err := DecodeCursor(raw)
		if err == nil {
			q.CursorTime = cursorTime
			q.CursorID = cursorID
		}
	}

	return q
}
