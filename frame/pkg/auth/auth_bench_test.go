package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"
)

func genRSAKeyForBench(b *testing.B) *rsa.PrivateKey {
	b.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		b.Fatalf("generate rsa key: %v", err)
	}
	return key
}

func BenchmarkGenerateTokenHS256(b *testing.B) {
	InitJWT(&JWTConfig{
		Secret:     "benchmark-test-secret-key-with-32-chars-123456",
		ExpireTime: time.Hour,
		Issuer:     "bench",
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GenerateToken(42, "bench-user")
	}
}

func BenchmarkParseTokenHS256(b *testing.B) {
	InitJWT(&JWTConfig{
		Secret:     "benchmark-test-secret-key-with-32-chars-123456",
		ExpireTime: time.Hour,
		Issuer:     "bench",
	})
	token, _ := GenerateToken(42, "bench-user")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseToken(token)
	}
}

func BenchmarkGenerateTokenRS256(b *testing.B) {
	priv := genRSAKeyForBench(b)
	InitJWT(&JWTConfig{
		SigningMethod: "RS256",
		PrivateKey:    priv,
		PublicKey:     &priv.PublicKey,
		ExpireTime:    time.Hour,
		Issuer:        "bench",
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GenerateToken(42, "bench-user")
	}
}
