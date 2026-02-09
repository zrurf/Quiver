package services

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/bytemare/opaque"
	"github.com/bytemare/opaque/message"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/zrurf/quiver/server/user/internal/dao"
)

const (
	accessTokenExpireSeconds  = 3600      // 1h有效期
	refreshTokenExpireSeconds = 86400 * 7 // 7d有效期
	maxTokenRetries           = 3         // 最大token生成重试次数
	sessionTokenLength        = 32        // token长度
)

type AuthService struct {
	userDao    *dao.UserRepository
	sessionDao *dao.SessionRepository
	imdb       *redis.Client
	opaque     *OpaqueService
}

func NewAuthService(
	userDao *dao.UserRepository,
	sessionDao *dao.SessionRepository,
	imdb *redis.Client,
	opaque *OpaqueService,
) *AuthService {
	return &AuthService{
		userDao:    userDao,
		sessionDao: sessionDao,
		imdb:       imdb,
		opaque:     opaque,
	}
}

// credentialIdentifierFromUsername 从用户名生成 credentialIdentifier
func (s *AuthService) credentialIdentifierFromUsername(username string) []byte {
	return []byte(username)
}

// UsernameExists 检测用户名是否存在
func (s *AuthService) UsernameExists(ctx context.Context, username string) (bool, error) {
	return s.userDao.UsernameExists(ctx, username)
}

// RegisterInit 处理注册第一步：接收 RegistrationRequest，返回 RegistrationResponse + serverPublicKey
func (s *AuthService) RegisterInit(ctx context.Context, username string, registrationRequest []byte) ([]byte, []byte, error) {
	// 反序列化客户端发来的 RegistrationRequest
	req, err := s.opaque.GetServer().Deserialize.RegistrationRequest(registrationRequest)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to deserialize registration request: %w", err)
	}

	credId := s.credentialIdentifierFromUsername(username)

	serverPubKey := s.opaque.GetServerPublicKey()

	resp, err := s.opaque.GetServer().RegistrationResponse(req, credId, nil)

	// 序列化并返回
	respBytes := resp.Serialize()
	return respBytes, serverPubKey, nil
}

// RegisterFinalize 处理注册第二步：接收 RegistrationRecord，存储到数据库
func (s *AuthService) RegisterFinalize(ctx context.Context, username string, registrationRecord []byte) error {
	// 反序列化验证格式，提前发现客户端错误
	record, err := s.opaque.GetServer().Deserialize.RegistrationRecord(registrationRecord)
	if err != nil {
		return fmt.Errorf("failed to deserialize registration record: %w", err)
	}
	_ = record // 回收
	// 将注册记录存到数据库 opaque_record 字段
	if err := s.userDao.SaveUserRecord(ctx, username, registrationRecord); err != nil {
		return fmt.Errorf("failed to save user opaque record: %w", err)
	}
	return nil
}

// LoginInit 处理登录第一步：接收 KE1，读取用户 record，返回 KE2
func (s *AuthService) LoginInit(ctx context.Context, username string, ke1Bytes []byte) ([]byte, []byte, []byte, error) {
	// 反序列化 KE1
	ke1, err := s.opaque.GetServer().Deserialize.KE1(ke1Bytes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to deserialize KE1: %w", err)
	}

	// 获取用户的 RegistrationRecord（opaque_record）
	_, recordBytes, err := s.userDao.GetUserRecord(ctx, username)
	if err != nil {
		// 为避免用户枚举，调用 conf.GetFakeRecord 构造假 record，继续返回 KE2
		log.Warn().Err(err).Msg("user not found")
		return s.fakeLoginInit(username, ke1)
	}

	// 反序列化 RegistrationRecord
	regRecord, err := s.opaque.GetServer().Deserialize.RegistrationRecord(recordBytes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to deserialize registration record: %w", err)
	}

	credId := s.credentialIdentifierFromUsername(username)

	// 构造 ClientRecord
	clientRecord := &opaque.ClientRecord{
		CredentialIdentifier: credId,
		ClientIdentity:       credId,
		RegistrationRecord:   regRecord,
	}

	// 4. 调用 LoginInit 得到 KE2
	ke2, output, err := s.opaque.GetServer().GenerateKE2(ke1, clientRecord)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("server login init failed: %w", err)
	}
	log.Debug().Any("ke1", base64.StdEncoding.EncodeToString(ke1.Serialize())).Any("ke2", base64.StdEncoding.EncodeToString(ke2.Serialize())).Msg("KE1 and KE2 generated")
	return ke2.Serialize(), output.SessionSecret, output.ClientMAC, nil
}

// 构造fake record
func (s *AuthService) fakeLoginInit(username string, ke1 *message.KE1) ([]byte, []byte, []byte, error) {
	record, err := s.opaque.conf.GetFakeRecord(s.credentialIdentifierFromUsername(username))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get fake record: %w", err)
	}
	ke2, output, err := s.opaque.GetServer().GenerateKE2(ke1, record)
	return ke2.Serialize(), output.SessionSecret, output.ClientMAC, err
}

// LoginFinalize 处理登录第二步：接收 KE3，校验 MAC，创建会话并返回 token + uid
func (s *AuthService) LoginFinalize(ctx context.Context, username string, ke3Bytes []byte, mac []byte) (int64, string, string, int64, error) {
	// 反序列化 KE3
	ke3, err := s.opaque.GetServer().Deserialize.KE3(ke3Bytes)
	if err != nil {
		return -1, "", "", -1, fmt.Errorf("failed to deserialize KE3: %w", err)
	}

	// 调用 LoginFinish 校验 MAC
	if err := s.opaque.GetServer().LoginFinish(ke3, mac); err != nil {
		return -1, "", "", -1, fmt.Errorf("login finish failed (invalid MAC): %w", err)
	}

	// 认证通过，获取用户 ID
	uid, _, err := s.userDao.GetUserRecord(ctx, username)
	if err != nil {
		return -1, "", "", -1, fmt.Errorf("failed to get user id: %w", err)
	}

	// 生成会话 token 并写入内存数据库
	accessToken, refreshToken, err := s.generateAndSaveToken(ctx, uid)
	if err != nil {
		return -1, "", "", -1, fmt.Errorf("failed to generate session token: %w", err)
	}

	// 更新最后登录时间
	_ = s.userDao.UpdateLastLogin(ctx, uid)

	return uid, accessToken, refreshToken, accessTokenExpireSeconds, nil
}

// 刷新token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (int64, string, string, int64, error) {
	if res, err := s.sessionDao.HasRefreshToken(ctx, refreshToken); err != nil || !res {
		return -1, "", "", -1, fmt.Errorf("refresh token not found")
	}
	uid, err := s.sessionDao.GetUidByRefreshToken(ctx, refreshToken)
	if err != nil {
		return -1, "", "", -1, fmt.Errorf("failed to get uid by refresh token: %w", err)
	}
	newAccessToken, newRefreshToken, err := s.generateAndSaveToken(ctx, uid)
	if err != nil {
		return -1, "", "", -1, fmt.Errorf("failed to generate new tokens: %w", err)
	}
	s.sessionDao.DelRefreshToken(ctx, refreshToken)
	return uid, newAccessToken, newRefreshToken, accessTokenExpireSeconds, nil
}

func (s *AuthService) generateAndSaveToken(ctx context.Context, uid int64) (string, string, error) {
	var accessToken, refreshToken string
	var accessOK = false
	var refreshOK = false
	for attempt := 0; attempt < maxTokenRetries; attempt++ {
		accessToken = s.opaque.GenerateToken(sessionTokenLength)
		if res, err := s.sessionDao.HasAccessToken(ctx, accessToken); err == nil && !res {
			accessOK = true
			break
		}
	}

	for attempt := 0; attempt < maxTokenRetries; attempt++ {
		refreshToken = s.opaque.GenerateToken(sessionTokenLength)
		if res, err := s.sessionDao.HasRefreshToken(ctx, refreshToken); err == nil && !res {
			refreshOK = true
			break
		}
	}

	if accessOK && refreshOK {
		err := s.sessionDao.SaveAccessToken(ctx, accessToken, uid, accessTokenExpireSeconds)
		if err != nil {
			log.Err(err).Msg("failed to save access token")
			return "", "", err
		}
		err = s.sessionDao.SaveRefreshToken(ctx, refreshToken, uid, refreshTokenExpireSeconds)
		if err != nil {
			s.sessionDao.DelAccessToken(ctx, accessToken)
			log.Err(err).Msg("failed to save refresh token")
			return "", "", err
		}
		return refreshToken, accessToken, nil
	} else {
		log.Warn().Msg("failed to generate unique tokens")
		return "", "", fmt.Errorf("failed to generate unique tokens")
	}
}
