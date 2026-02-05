package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"
	"github.com/zrurf/quiver/server/user/internal/api"
	"github.com/zrurf/quiver/server/user/internal/model"
	"github.com/zrurf/quiver/server/user/internal/services"
)

type AuthHandler struct {
	auth *services.AuthService
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{
		auth: auth,
	}
}

// RegisterInit 对应 /api/auth/register-init
func (h *AuthHandler) RegisterInit(c fiber.Ctx) error {
	var req model.RegisterInitRequest
	if err := c.Bind().Body(&req); err != nil {
		log.Error().Any("ctx", c).Err(err).Msg("bind request body failed")
		return api.Error(c, fiber.StatusBadRequest, api.CodeInvalidBody, "invalid request body", api.StatusErrInvalidBody)
	}
	if req.Username == "" || len(req.RegistrationRequest) == 0 {
		log.Info().Any("ctx", c).Msg("missing username or registrationRequest")
		return api.Error(c, fiber.StatusBadRequest, api.CodeInvalidBody, "missing username or registrationRequest", api.StatusErrInvalidBody)
	}

	respBytes, pubKey, err := h.auth.RegisterInit(c.Context(), req.Username, req.RegistrationRequest)
	if err != nil {
		log.Error().Any("ctx", c).Err(err).Msg("register init failed")
		return api.Error(c, fiber.StatusInternalServerError, api.CodeServerError, "register init failed", api.StatusErrServer)
	}

	log.Info().Str("username", req.Username).Msg("register init OK")
	return api.Success(c, model.RegisterInitResponse{
		RegistrationResponse: respBytes,
		ServerPublicKey:      pubKey,
	}, "register init OK")
}

// RegisterFinalize 对应 /api/auth/register-finalize
func (h *AuthHandler) RegisterFinalize(c fiber.Ctx) error {
	var req model.RegisterFinalizeRequest
	if err := c.Bind().Body(&req); err != nil {
		log.Error().Any("ctx", c).Err(err).Msg("bind request body failed")
		return api.Error(c, fiber.StatusBadRequest, api.CodeInvalidBody, "invalid request body", api.StatusErrInvalidBody)
	}
	if req.Username == "" || len(req.RegistrationRecord) == 0 {
		log.Info().Any("ctx", c).Msg("missing username or registrationRecord")
		return api.Error(c, fiber.StatusBadRequest, api.CodeInvalidBody, "missing username or registrationRecord", api.StatusErrInvalidBody)
	}

	if err := h.auth.RegisterFinalize(c.Context(), req.Username, req.RegistrationRecord); err != nil {
		log.Error().Any("ctx", c).Err(err).Msg("register finalize failed")
		return api.Error(c, fiber.StatusInternalServerError, api.CodeServerError, "register finalize failed", api.StatusErrServer)
	}

	log.Info().Str("username", req.Username).Msg("register finalize OK")
	return api.Success(c, model.RegisterFinalizeResponse{OK: true}, "register finalize OK")
}

// LoginInit 对应 /api/auth/login-init
func (h *AuthHandler) LoginInit(c fiber.Ctx) error {
	var req model.LoginInitRequest
	if err := c.Bind().Body(&req); err != nil {
		log.Error().Any("ctx", c).Err(err).Msg("bind request body failed")
		return api.Error(c, fiber.StatusBadRequest, api.CodeInvalidBody, "invalid request body", api.StatusErrInvalidBody)
	}
	if req.Username == "" || len(req.KE1) == 0 {
		log.Info().Any("ctx", c).Msg("missing username or ke1")
		return api.Error(c, fiber.StatusBadRequest, api.CodeInvalidBody, "missing username or ke1", api.StatusErrInvalidBody)
	}

	ke2Bytes, err := h.auth.LoginInit(c.Context(), req.Username, req.KE1)
	if err != nil {
		log.Error().Any("ctx", c).Err(err).Msg("login init failed")
		return api.Error(c, fiber.StatusInternalServerError, api.CodeServerError, "login init failed", api.StatusErrServer)
	}

	log.Info().Str("username", req.Username).Msg("login init OK")
	return api.Success(c, model.LoginInitResponse{KE2: ke2Bytes}, "login init OK")
}

// LoginFinalize 对应 /api/auth/login-finalize
func (h *AuthHandler) LoginFinalize(c fiber.Ctx) error {
	var req model.LoginFinalizeRequest
	if err := c.Bind().Body(&req); err != nil {
		log.Error().Any("ctx", c).Err(err).Msg("bind request body failed")
		return api.Error(c, fiber.StatusBadRequest, api.CodeInvalidBody, "invalid request body", api.StatusErrInvalidBody)
	}
	if len(req.KE3) == 0 {
		log.Info().Any("ctx", c).Msg("missing ke3")
		return api.Error(c, fiber.StatusBadRequest, api.CodeInvalidBody, "missing ke3", api.StatusErrInvalidBody)
	}

	uid, token, err := h.auth.LoginFinalize(c.Context(), "", req.KE3)
	if err != nil {
		log.Error().Any("ctx", c).Err(err).Msg("login finalize failed")
		return api.Error(c, fiber.StatusUnauthorized, 602, "login finalize failed", "ERR_LOGIN_FAILED")
	}

	log.Info().Str("KE3", string(req.KE3)).Int64("uid", uid).Msg("login finalize OK")
	return api.Success(c, model.LoginFinalizeResponse{
		UID:   uid,
		Token: token,
	}, "login finalize OK")
}
