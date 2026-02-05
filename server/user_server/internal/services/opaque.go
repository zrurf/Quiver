// services/opaque.go
package services

import (
	"fmt"

	"github.com/bytemare/opaque"
)

const (
	SessionTokenLength = 32
)

type OpaqueService struct {
	conf         *opaque.Configuration
	server       *opaque.Server
	serverPubKey []byte // AKE public key
	oprfSeed     []byte
}

type OpaqueConfig struct {
	Config          *opaque.Configuration
	OprfSeed        []byte
	ServerPublicKey []byte
	ServerSecretKey []byte
}

// NewOpaqueService 创建并初始化 OpaqueService。
func NewOpaqueService(config *OpaqueConfig) (*OpaqueService, error) {
	server, err := opaque.NewServer(config.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create OPAQUE server: %w", err)
	}

	err = server.SetKeyMaterial(
		nil,
		config.ServerSecretKey,
		config.ServerPublicKey,
		config.OprfSeed,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set OPAQUE key material: %w", err)
	}

	return &OpaqueService{
		conf:         config.Config,
		server:       server,
		serverPubKey: config.ServerPublicKey,
		oprfSeed:     config.OprfSeed,
	}, nil
}

// GetServerPublicKey 返回服务器 AKE 公钥（发往客户端，用于注册与登录）。
func (s *OpaqueService) GetServerPublicKey() []byte {
	return s.serverPubKey
}

// GetOPRFSeed 返回 OPRF 种子（供 Register/Login 流程中按用户名/credentialIdentifier 派生 OPRF key）。
func (s *OpaqueService) GetOPRFSeed() []byte {
	return s.oprfSeed
}

// GetConfig 返回配置对象，方便构造反序列化器（如需要自行解析客户端消息）。
func (s *OpaqueService) GetConfig() *opaque.Configuration {
	return s.conf
}

// GetServer 返回底层的 opaque.Server，供 AuthOpaqueService 调用 RegisterResponse / LoginInit / LoginFinish。
func (s *OpaqueService) GetServer() *opaque.Server {
	return s.server
}

// GenerateSessionToken 生成 32 字节随机 token，用作 Opaque Token。
func (s *OpaqueService) GenerateSessionToken() (string, error) {
	b := opaque.RandomBytes(SessionTokenLength)
	return fmt.Sprintf("%x", b), nil
}
