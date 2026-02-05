package services

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/bytemare/opaque"
	"github.com/bytemare/opaque/message"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/zrurf/quiver/server/user/internal/dao"
)

const (
	sessionKeyPrefix     = "session:"
	sessionExpireSeconds = 86400 // 单位秒，24h
)

type AuthService struct {
	dao    *dao.UserRepository
	imdb   *redis.Client
	opaque *OpaqueService
}

func NewAuthService(
	dao *dao.UserRepository,
	imdb *redis.Client,
	opaque *OpaqueService,
) *AuthService {
	return &AuthService{
		dao:    dao,
		imdb:   imdb,
		opaque: opaque,
	}
}

// credentialIdentifierFromUsername 从用户名生成 credentialIdentifier
func (s *AuthService) credentialIdentifierFromUsername(username string) []byte {
	return []byte(username)
}

// RegisterInit 处理注册第一步：接收 RegistrationRequest，返回 RegistrationResponse + serverPublicKey
func (s *AuthService) RegisterInit(ctx context.Context, username string, registrationRequest []byte) ([]byte, []byte, error) {
	// 反序列化客户端发来的 RegistrationRequest
	req, err := s.opaque.GetServer().Deserialize.RegistrationRequest(registrationRequest)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to deserialize registration request: %w", err)
	}

	credId := s.credentialIdentifierFromUsername(username)
	oprfSeed := s.opaque.GetOPRFSeed()
	serverPubKey := s.opaque.GetServerPublicKey()

	// 将 serverPublicKey 从 []byte 转为 group.Element
	pksElement := s.opaque.server.GetConf().Group.NewElement()
	if err := pksElement.Decode(serverPubKey); err != nil {
		return nil, nil, fmt.Errorf("failed to decode server public key: %w", err)
	}

	// 生成 RegistrationResponse
	resp := s.opaque.GetServer().RegistrationResponse(req, pksElement, credId, oprfSeed)

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
	if err := s.dao.SaveUserRecord(ctx, username, registrationRecord); err != nil {
		return fmt.Errorf("failed to save user opaque record: %w", err)
	}
	return nil
}

// LoginInit 处理登录第一步：接收 KE1，读取用户 record，返回 KE2
func (s *AuthService) LoginInit(ctx context.Context, username string, ke1Bytes []byte) ([]byte, error) {
	// 反序列化 KE1
	ke1, err := s.opaque.GetServer().Deserialize.KE1(ke1Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize KE1: %w", err)
	}

	// 获取用户的 RegistrationRecord（opaque_record）
	_, recordBytes, err := s.dao.GetUserRecord(ctx, username)
	if err != nil {
		// 为避免用户枚举，调用 conf.GetFakeRecord 构造假 record，继续返回 KE2
		log.Warn().Err(err).Msg("user not found")
		return s.fakeLoginInit(username, ke1)
	}

	// 反序列化 RegistrationRecord
	regRecord, err := s.opaque.GetServer().Deserialize.RegistrationRecord(recordBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize registration record: %w", err)
	}

	credId := s.credentialIdentifierFromUsername(username)
	// 构造 ClientRecord
	clientRecord := &opaque.ClientRecord{
		CredentialIdentifier: credId,
		ClientIdentity:       []byte("quiver"),
		RegistrationRecord:   regRecord,
		TestMaskNonce:        nil,
	}

	// 4. 调用 LoginInit 得到 KE2
	ke2, err := s.opaque.GetServer().LoginInit(ke1, clientRecord)
	if err != nil {
		return nil, fmt.Errorf("server login init failed: %w", err)
	}
	log.Debug().Any("ke1", base64.StdEncoding.EncodeToString(ke1.Serialize())).Any("ke2", base64.StdEncoding.EncodeToString(ke2.Serialize())).Msg("KE1 and KE2 generated")
	return ke2.Serialize(), nil
}

// 构造fake record
func (s *AuthService) fakeLoginInit(username string, ke1 *message.KE1) ([]byte, error) {
	record, err := s.opaque.conf.GetFakeRecord(s.credentialIdentifierFromUsername(username))
	if err != nil {
		return nil, fmt.Errorf("failed to get fake record: %w", err)
	}
	ke2, err := s.opaque.GetServer().LoginInit(ke1, record)
	return ke2.Serialize(), err
}

// LoginFinalize 处理登录第二步：接收 KE3，校验 MAC，创建会话并返回 token + uid
func (s *AuthService) LoginFinalize(ctx context.Context, username string, ke3Bytes []byte) (int64, string, error) {
	// 反序列化 KE3
	ke3, err := s.opaque.GetServer().Deserialize.KE3(ke3Bytes)
	if err != nil {
		return 0, "", fmt.Errorf("failed to deserialize KE3: %w", err)
	}

	// 调用 LoginFinish 校验 MAC
	if err := s.opaque.GetServer().LoginFinish(ke3); err != nil {
		return 0, "", fmt.Errorf("login finish failed (invalid MAC): %w", err)
	}

	// 认证通过，获取用户 ID
	uid, _, err := s.dao.GetUserRecord(ctx, username)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get user id: %w", err)
	}

	// 生成会话 token 并写入内存数据库
	token, err := s.opaque.GenerateSessionToken()
	if err != nil {
		return 0, "", fmt.Errorf("failed to generate session token: %w", err)
	}

	key := sessionKeyPrefix + token

	// 将 UID 写入，并设置过期时间
	if err := s.imdb.Set(ctx, key, uid, time.Second*time.Duration(sessionExpireSeconds)).Err(); err != nil {
		return 0, "", fmt.Errorf("failed to store session token: %w", err)
	}

	// 更新最后登录时间
	_ = s.dao.UpdateLastLogin(ctx, uid)

	return uid, token, nil
}

// GetUIDBySession 从内存数据库中根据 token 获取 UID
func (s *AuthService) GetUIDBySession(ctx context.Context, token string) (int64, error) {
	key := sessionKeyPrefix + token
	val, err := s.imdb.Get(ctx, key).Result() // 从 Redis 中取出的字符串形式的 UID
	if err != nil {
		return 0, err
	}
	uid := int64(binary.BigEndian.Uint64([]byte(val))) // UID转类型
	return uid, nil
}
