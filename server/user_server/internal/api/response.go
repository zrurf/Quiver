package api

import (
	"time"

	"github.com/gofiber/fiber/v3"
)

func Success(c fiber.Ctx, payload any, msg string) error {
	return c.Status(fiber.StatusOK).JSON(ResponseModel{
		Code:      0,
		Message:   msg,
		Status:    "OK",
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	})
}

func Error(c fiber.Ctx, httpCode int, code int, msg string, status string) error {
	return c.Status(httpCode).JSON(ResponseModel{
		Code:      code,
		Message:   msg,
		Status:    status,
		Timestamp: time.Now().UnixMilli(),
	})
}
