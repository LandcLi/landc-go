package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

// EncPrefix 加密结果的前缀标记。
// Decrypt 遇此前缀走 AES-GCM 解密；无此前缀的数据视为明文原样返回（兼容历史明文存储）。
const EncPrefix = "enc:"

// Cipher AES-GCM 加解密器。
//
// 密钥由调用方显式传入（环境变量读取属于业务/配置层职责，本包不绑定任何配置来源）。
type Cipher struct {
	aead cipher.AEAD
}

// NewCipher 使用 16/24/32 字节密钥创建 AES-GCM 加解密器。
func NewCipher(key []byte) (*Cipher, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, fmt.Errorf("security: AES key must be 16/24/32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("security: create AES cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("security: create GCM: %w", err)
	}
	return &Cipher{aead: aead}, nil
}

// NewCipherFromString 使用密钥字符串创建加解密器。
// 优先按 hex 解码（如 32 字节 hex 密钥），失败则按原始字节使用（须 16/24/32 字节）。
func NewCipherFromString(key string) (*Cipher, error) {
	if decoded, err := hex.DecodeString(key); err == nil {
		return NewCipher(decoded)
	}
	return NewCipher([]byte(key))
}

// Encrypt 加密明文，返回带 enc: 前缀的 base64 密文。
// 每次加密使用 crypto/rand 随机 nonce；结果格式：enc: base64(nonce || sealed)。
func (c *Cipher) Encrypt(plaintext []byte) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("security: generate nonce: %w", err)
	}
	sealed := c.aead.Seal(nil, nonce, plaintext, nil)
	raw := make([]byte, 0, len(nonce)+len(sealed))
	raw = append(raw, nonce...)
	raw = append(raw, sealed...)
	return EncPrefix + base64.StdEncoding.EncodeToString(raw), nil
}

// Decrypt 解密：带 enc: 前缀走 AES-GCM；无前缀视为明文原样返回。
func (c *Cipher) Decrypt(data string) ([]byte, error) {
	if !strings.HasPrefix(data, EncPrefix) {
		return []byte(data), nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, EncPrefix))
	if err != nil {
		return nil, fmt.Errorf("security: decode ciphertext: %w", err)
	}
	nonceSize := c.aead.NonceSize()
	if len(raw) < nonceSize {
		return nil, errors.New("security: ciphertext too short")
	}
	nonce, sealed := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := c.aead.Open(nil, nonce, sealed, nil)
	if err != nil {
		return nil, fmt.Errorf("security: decrypt: %w", err)
	}
	return plaintext, nil
}
