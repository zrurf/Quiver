// services/opaque.go
package services

import (
	"encoding/hex"
	"fmt"

	"github.com/bytemare/ecc"
	"github.com/bytemare/opaque"
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

	privateKey, err := opaque.DeserializeScalar(ecc.Ristretto255Sha512, config.ServerSecretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize server secret key: %w", err)
	}

	err = server.SetKeyMaterial(&opaque.ServerKeyMaterial{
		PrivateKey:     privateKey,
		PublicKeyBytes: config.ServerPublicKey,
		OPRFGlobalSeed: config.OprfSeed,
		Identity:       []byte("quiver"),
	})

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

// GenerateToken 生成指定长度随机token
func (s *OpaqueService) GenerateToken(length int) string {
	return hex.EncodeToString(opaque.RandomBytes(length))
}
