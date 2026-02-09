package handler

import (
	"time"

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
		return api.Error(c, fiber.StatusBadRequest, api.CodeInvalidBody, "missing username or registrationRequest", api.StatusErrInvalidBody)
	}

	if exists, err := h.auth.UsernameExists(c.Context(), req.Username); err != nil {
		log.Error().Any("ctx", c).Err(err).Msg("failed to check username exists")
		return api.Error(c, fiber.StatusInternalServerError, api.CodeServerError, "failed to check username exists", api.StatusErrServer)
	} else if exists {
		return api.Error(c, fiber.StatusBadRequest, api.CodeRegisterInitFailed, "username already exists", api.StatusErrUsernameConflict)
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
		return api.Error(c, fiber.StatusBadRequest, api.CodeInvalidBody, "missing username or ke1", api.StatusErrInvalidBody)
	}

	if exists, err := h.auth.UsernameExists(c.Context(), req.Username); err != nil {
		log.Error().Any("ctx", c).Err(err).Msg("failed to check username exists")
		return api.Error(c, fiber.StatusInternalServerError, api.CodeServerError, "failed to check username exists", api.StatusErrServer)
	} else if !exists {
		return api.Error(c, fiber.StatusBadRequest, api.CodeLoginFailed, "login init failed", api.StatusErrLogin)
	}

	ke2Bytes, _, clientMAC, err := h.auth.LoginInit(c.Context(), req.Username, req.KE1)
	if err != nil {
		log.Error().Any("ctx", c).Err(err).Msg("login init failed")
		return api.Error(c, fiber.StatusInternalServerError, api.CodeServerError, "login init failed", api.StatusErrServer)
	}

	return api.Success(c, model.LoginInitResponse{KE2: ke2Bytes, MAC: clientMAC}, "login init OK")
}

// LoginFinalize 对应 /api/auth/login-finalize
func (h *AuthHandler) LoginFinalize(c fiber.Ctx) error {
	var req model.LoginFinalizeRequest
	if err := c.Bind().Body(&req); err != nil {
		log.Error().Any("ctx", c).Err(err).Msg("bind request body failed")
		return api.Error(c, fiber.StatusBadRequest, api.CodeInvalidBody, "invalid request body", api.StatusErrInvalidBody)
	}
	if len(req.KE3) == 0 {
		return api.Error(c, fiber.StatusBadRequest, api.CodeInvalidBody, "missing ke3", api.StatusErrInvalidBody)
	}

	uid, accessToken, refreshToken, expire, err := h.auth.LoginFinalize(c.Context(), req.Username, req.KE3, req.MAC)
	if err != nil {
		log.Error().Any("ctx", c).Err(err).Msg("login finalize failed")
		return api.Error(c, fiber.StatusUnauthorized, api.CodeLoginFailed, "login finalize failed", api.StatusErrLogin)
	}

	log.Info().Str("KE3", string(req.KE3)).Int64("uid", uid).Msg("login finalize OK")
	return api.Success(c, model.LoginFinalizeResponse{
		UID:          uid,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpireAt:     time.Now().UnixMilli() + (expire * 1000),
	}, "login finalize OK")
}

// RefreshToken 刷新令牌
func (h *AuthHandler) RefreshToken(c fiber.Ctx) error {
	var req model.RefreshTokenRequest
	if err := c.Bind().Body(&req); err != nil {
		log.Error().Any("ctx", c).Err(err).Msg("bind request body failed")
		return api.Error(c, fiber.StatusBadRequest, api.CodeInvalidBody, "invalid request body", api.StatusErrInvalidBody)
	}
	if len(req.RefreshToken) == 0 {
		return api.Error(c, fiber.StatusBadRequest, api.CodeInvalidBody, "missing refresh token", api.StatusErrInvalidBody)
	}
	uid, accessToken, refreshToken, expire, err := h.auth.RefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		log.Error().Any("ctx", c).Err(err).Msg("refresh token failed")
		return api.Error(c, fiber.StatusUnauthorized, api.CodeRefreshFailed, "refresh token failed", api.StatusErrRefresh)
	}
	log.Info().Str("token", string(req.RefreshToken)).Int64("uid", uid).Msg("refresh token successful")
	return api.Success(c, model.LoginFinalizeResponse{
		UID:          uid,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpireAt:     time.Now().UnixMilli() + (expire * 1000),
	}, "refresh token successful")
}
