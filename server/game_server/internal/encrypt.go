package internal

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// Encryptor 管理固定密钥
type Encryptor struct {
	fixedKey []byte // 固定32字节密钥
}

// NewEncryptor 从配置创建加密器，使用固定密钥
func NewEncryptor(fixedKeyBase64 string) (*Encryptor, error) {
	var fixedKey []byte
	if fixedKeyBase64 != "" {
		key, err := base64.StdEncoding.DecodeString(fixedKeyBase64)
		if err != nil {
			return nil, err
		}
		if len(key) != 32 {
			return nil, errors.New("fixed key must be 32 bytes")
		}
		fixedKey = key
	} else {
		fixedKey = make([]byte, 32)
	}
	return &Encryptor{fixedKey: fixedKey}, nil
}

// GetRoomKey 返回固定密钥（所有房间相同，仅供演示，生产时别用）
func (e *Encryptor) GetRoomKey() []byte {
	return e.fixedKey
}

// Encrypt 使用 AES-GCM 加密数据，密钥由调用方提供
func Encrypt(plain []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plain, nil), nil
}

// Decrypt 使用 AES-GCM 解密
func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
