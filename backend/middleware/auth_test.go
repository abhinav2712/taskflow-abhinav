package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestAuthenticateRejectsMissingBearerToken(t *testing.T) {
	t.Parallel()

	handler := Authenticate("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called without bearer token")
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	if recorder.Body.String() != "{\"error\":\"unauthorized\"}\n" {
		t.Fatalf("unexpected unauthorized body: %q", recorder.Body.String())
	}
}

func TestAuthenticateInjectsUserIntoContext(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID.String(),
		"email":   "test@example.com",
	})

	tokenString, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	handler := Authenticate("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := UserIDFromCtx(r.Context()); got != userID {
			t.Fatalf("expected user id %s, got %s", userID, got)
		}

		if got := EmailFromCtx(r.Context()); got != "test@example.com" {
			t.Fatalf("expected email test@example.com, got %q", got)
		}

		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+tokenString)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestAuthenticateRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	handler := Authenticate("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called for invalid token")
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}
