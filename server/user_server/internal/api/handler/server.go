package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/zrurf/quiver/server/user/internal/api"
)

func HandleServerStatus(c fiber.Ctx) error {
	return api.Success(c, nil, "OK")
}
