package utils

import (
	"testing"
	"time"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("super-secret")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "super-secret" {
		t.Fatal("plaintext")
	}
	if !CheckPassword(hash, "super-secret") {
		t.Fatal("check failed")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("should fail")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key := "my-app-encryption-key"
	enc, err := Encrypt("registry-password", key)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := Decrypt(enc, key)
	if err != nil {
		t.Fatal(err)
	}
	if dec != "registry-password" {
		t.Fatalf("got %q", dec)
	}
}

func TestJWT(t *testing.T) {
	tok, err := GenerateToken(1, "admin", "ADMIN", "secret", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseToken(tok, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if claims.Username != "admin" || claims.Role != "ADMIN" {
		t.Fatalf("claims: %+v", claims)
	}
}
