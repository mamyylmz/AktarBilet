package auth

import (
	"testing"

	"github.com/google/uuid"
	"github.com/mamyylmz/AktarBilet/internal/domain"
)

func TestAuth_PasswordHashAndVerify(t *testing.T) {
	svc := NewAuthService("secret")
	pass := "StrongPassword123!"

	hash, err := svc.HashPassword(pass)
	if err != nil {
		t.Fatalf("Şifre özetleme hatası: %v", err)
	}

	if !svc.CheckPassword(hash, pass) {
		t.Fatalf("Doğru şifre doğrulanamadı!")
	}
	if svc.CheckPassword(hash, "WrongPassword") {
		t.Fatalf("Yanlış şifre başarılı kabul edildi!")
	}
}

func TestAuth_JWTGenerationAndValidation(t *testing.T) {
	svc := NewAuthService("test-jwt-secret-key-1234567890123456")
	user := &domain.User{
		ID:       uuid.New(),
		Email:    "kontrolor@aktarbilet.gov.tr",
		FullName: "Ahmet Kontrolör",
		Role:     domain.RoleKontrolor,
	}

	access, refresh, err := svc.GenerateTokens(user)
	if err != nil {
		t.Fatalf("Token üretim hatası: %v", err)
	}

	if access == "" || refresh == "" {
		t.Fatalf("Boş token üretildi")
	}

	claims, err := svc.ValidateToken(access)
	if err != nil {
		t.Fatalf("Token doğrulama hatası: %v", err)
	}

	if claims.UserID != user.ID || claims.Role != domain.RoleKontrolor {
		t.Fatalf("Token claims uyuşmuyor: %+v", claims)
	}

	// Geçersiz token testi
	_, err = svc.ValidateToken("invalid.jwt.token")
	if err != ErrInvalidToken {
		t.Fatalf("Geçersiz token reddedilmeliydi, alınan hata: %v", err)
	}
}
