package auth_test

import (
	"testing"
	"time"

	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/auth"
)

func TestGenerateToken_Success(t *testing.T) {
	secret := "test-secret-key"
	userID := "user-123"
	deviceID := "device-456"

	token, err := auth.GenerateToken(secret, userID, deviceID, 1*time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("GenerateToken() returned empty token")
	}
}

func TestValidateToken_Valid(t *testing.T) {
	secret := "test-secret-key"
	userID := "user-123"
	deviceID := "device-456"

	token, _ := auth.GenerateToken(secret, userID, deviceID, 1*time.Hour)

	claims, err := auth.ValidateToken(secret, token)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("claims.UserID = %q, want %q", claims.UserID, userID)
	}
	if claims.DeviceID != deviceID {
		t.Errorf("claims.DeviceID = %q, want %q", claims.DeviceID, deviceID)
	}
}

func TestValidateToken_Expired(t *testing.T) {
	secret := "test-secret-key"
	token, _ := auth.GenerateToken(secret, "user-1", "device-1", -1*time.Hour)

	_, err := auth.ValidateToken(secret, token)
	if err == nil {
		t.Fatal("ValidateToken() expected error for expired token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	token, _ := auth.GenerateToken("secret-a", "user-1", "device-1", 1*time.Hour)
	_, err := auth.ValidateToken("secret-b", token)
	if err == nil {
		t.Fatal("ValidateToken() expected error for wrong secret")
	}
}

func TestValidateToken_InvalidFormat(t *testing.T) {
	_, err := auth.ValidateToken("secret", "not-a-jwt-token")
	if err == nil {
		t.Fatal("ValidateToken() expected error for invalid format")
	}
}

func TestHashPassword_And_Compare(t *testing.T) {
	password := "my-password-123"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword() returned empty hash")
	}
	if hash == password {
		t.Fatal("HashPassword() returned plaintext")
	}
	if !auth.CheckPassword(hash, password) {
		t.Error("CheckPassword() = false, want true")
	}
	if auth.CheckPassword(hash, "wrong-password") {
		t.Error("CheckPassword() = true for wrong password, want false")
	}
}
