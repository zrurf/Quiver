package internal

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/zrurf/quiver/server/user/internal/api/handler"
	"github.com/zrurf/quiver/server/user/internal/services"
)

type RouteDependencies struct {
	AuthSvc *services.AuthService
}

func ConfigRoute(app *fiber.App, dep *RouteDependencies) {

	authHandler := handler.NewAuthHandler(dep.AuthSvc)

	app.All("/", handler.HandleServerStatus)
	app.All("/health", healthcheck.New())

	// 注册接口
	app.Post("/api/auth/register-init", authHandler.RegisterInit)
	app.Post("/api/auth/register-finalize", authHandler.RegisterFinalize)

	// 登录接口
	app.Post("/api/auth/login-init", authHandler.LoginInit)
	app.Post("/api/auth/login-finalize", authHandler.LoginFinalize)
}
