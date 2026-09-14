package main

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model"
)

func ok(c *fiber.Ctx, message string, data any) error {
	return c.JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(model.WebResponse{
		Success: false,
		Message: message,
	})
}

func noContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

func paramID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return 0, false
	}

	return id, true
}

func parseListQuery(c *fiber.Ctx) model.ListQuery {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	if page < 1 {
		page = 1
	}

	if limit < 1 {
		limit = 10
	}

	if limit > 100 {
		limit = 100
	}

	order := strings.ToLower(c.Query("order", "asc"))

	if order != "asc" && order != "desc" {
		order = "asc"
	}

	sort := c.Query("sort", "id")

	allowedSort := map[string]bool{
		"id":         true,
		"username":   true,
		"email":      true,
		"is_active":  true,
		"created_at": true,
	}

	if !allowedSort[sort] {
		sort = "id"
	}

	var isActive *bool

	if value := c.Query("is_active"); value != "" {
		if value == "true" {
			v := true
			isActive = &v
		} else if value == "false" {
			v := false
			isActive = &v
		}
	}

	return model.ListQuery{
		Page:     page,
		Limit:    limit,
		Search:   c.Query("search"),
		Sort:     sort,
		Order:    order,
		IsActive: isActive,
	}
}

func failValidation(c *fiber.Ctx, errs any) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(model.WebResponse{
		Success: false,
		Message: "validasi gagal",
		Errors:  errs,
	})
}

func created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location)

	return c.Status(fiber.StatusCreated).JSON(model.WebResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}
