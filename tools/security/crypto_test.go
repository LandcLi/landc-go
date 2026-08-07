package security

import (
	"strings"
	"testing"
)

func TestNewCipherKeyLength(t *testing.T) {
	for _, n := range []int{16, 24, 32} {
		if _, err := NewCipher(make([]byte, n)); err != nil {
			t.Errorf("key length %d should be valid: %v", n, err)
		}
	}
	if _, err := NewCipher(make([]byte, 20)); err == nil {
		t.Error("key length 20 should error")
	}
	if _, err := NewCipher(nil); err == nil {
		t.Error("nil key should error")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	c, err := NewCipher([]byte("0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	enc, err := c.Encrypt([]byte("hello world"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(enc, EncPrefix) {
		t.Errorf("encrypted should have %q prefix, got %q", EncPrefix, enc)
	}
	dec, err := c.Decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	if string(dec) != "hello world" {
		t.Errorf("round trip = %q, want %q", dec, "hello world")
	}
}

func TestEncryptRandomness(t *testing.T) {
	// 相同明文两次加密结果应不同（随机 nonce）
	c, _ := NewCipher([]byte("0123456789abcdef"))
	e1, _ := c.Encrypt([]byte("same"))
	e2, _ := c.Encrypt([]byte("same"))
	if e1 == e2 {
		t.Error("encryptions should differ (random nonce)")
	}
}

func TestDecryptPlaintextFallback(t *testing.T) {
	c, _ := NewCipher([]byte("0123456789abcdef"))
	// 无 enc: 前缀：视为明文原样返回（兼容历史明文存储）
	dec, err := c.Decrypt("plain-text")
	if err != nil {
		t.Fatal(err)
	}
	if string(dec) != "plain-text" {
		t.Errorf("fallback = %q, want %q", dec, "plain-text")
	}
}

func TestDecryptTampered(t *testing.T) {
	c, _ := NewCipher([]byte("0123456789abcdef"))
	enc, _ := c.Encrypt([]byte("secret"))
	// 篡改最后一个 base64 字符（仍为合法 base64，GCM 认证应失败）
	tampered := enc[:len(enc)-1] + "x"
	if _, err := c.Decrypt(tampered); err == nil {
		t.Error("tampered ciphertext should fail GCM auth")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	c1, _ := NewCipher([]byte("0123456789abcdef"))
	c2, _ := NewCipher([]byte("fedcba9876543210"))
	enc, _ := c1.Encrypt([]byte("secret"))
	if _, err := c2.Decrypt(enc); err == nil {
		t.Error("wrong key should fail decryption")
	}
}

func TestNewCipherFromStringHex(t *testing.T) {
	keyHex := "0123456789abcdef0123456789abcdef" // 32 hex chars = 16 bytes
	c, err := NewCipherFromString(keyHex)
	if err != nil {
		t.Fatalf("hex key should work: %v", err)
	}
	enc, err := c.Encrypt([]byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	if dec, err := c.Decrypt(enc); err != nil || string(dec) != "x" {
		t.Errorf("hex key round trip = %q, %v", dec, err)
	}
}
