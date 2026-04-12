package handler

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func TestHashPasswordUsesBcryptCost12(t *testing.T) {
	t.Parallel()

	hash, err := hashPassword("password123")
	if err != nil {
		t.Fatalf("hashPassword returned error: %v", err)
	}

	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("bcrypt.Cost returned error: %v", err)
	}

	if cost != bcryptCost {
		t.Fatalf("expected bcrypt cost %d, got %d", bcryptCost, cost)
	}
}

func TestComparePassword(t *testing.T) {
	t.Parallel()

	hash, err := hashPassword("password123")
	if err != nil {
		t.Fatalf("hashPassword returned error: %v", err)
	}

	if err := comparePassword(hash, "password123"); err != nil {
		t.Fatalf("expected matching password to validate: %v", err)
	}

	if err := comparePassword(hash, "wrong-password"); err == nil {
		t.Fatal("expected wrong password to fail")
	}
}

func TestGenerateTokenIncludesRequiredClaims(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	secret := "supersecret"

	tokenString, err := generateToken(userID, "test@example.com", secret)
	if err != nil {
		t.Fatalf("generateToken returned error: %v", err)
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("jwt.Parse returned error: %v", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("expected jwt.MapClaims")
	}

	if claims["user_id"] != userID.String() {
		t.Fatalf("expected user_id %q, got %#v", userID.String(), claims["user_id"])
	}

	if claims["email"] != "test@example.com" {
		t.Fatalf("expected email claim, got %#v", claims["email"])
	}

	expFloat, ok := claims["exp"].(float64)
	if !ok {
		t.Fatalf("expected numeric exp claim, got %#v", claims["exp"])
	}

	expiration := time.Unix(int64(expFloat), 0)
	minExpected := time.Now().Add(23 * time.Hour)
	maxExpected := time.Now().Add(25 * time.Hour)

	if expiration.Before(minExpected) || expiration.After(maxExpected) {
		t.Fatalf("expected exp around 24h from now, got %v", expiration)
	}
}
