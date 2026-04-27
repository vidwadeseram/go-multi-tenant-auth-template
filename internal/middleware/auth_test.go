package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/config"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/services"
)

const testJWTSecret = "test-secret-key-for-unit-tests"

func testSettings() *config.Settings {
	return &config.Settings{
		JWTSecret:              testJWTSecret,
		JWTAccessExpireMinutes: 15,
		JWTRefreshExpireDays:   7,
	}
}

func issueTestAccessToken(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":  userID.String(),
		"type": "access",
		"exp":  time.Now().Add(15 * time.Minute).Unix(),
		"iat":  time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}
	return signed
}

func runAuthMiddleware(t *testing.T, authorizationHeader string) *httptest.ResponseRecorder {
	t.Helper()
	tokenSvc := services.NewTokenService(testSettings())
	mw := NewAuthMiddleware(tokenSvc, nil)

	router := gin.New()
	router.Use(mw.Handle())
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if authorizationHeader != "" {
		req.Header.Set("Authorization", authorizationHeader)
	}
	router.ServeHTTP(w, req)
	return w
}

func TestAuthMiddleware_MissingAuthorizationHeader_Returns401(t *testing.T) {
	w := runAuthMiddleware(t, "")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	var body map[string]map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}
	if body["error"]["code"] != "AUTHENTICATION_REQUIRED" {
		t.Errorf("unexpected error code: %s", body["error"]["code"])
	}
}

func TestAuthMiddleware_MalformedHeader_Returns401(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"no bearer prefix", "justtoken"},
		{"wrong scheme", "Basic dXNlcjpwYXNz"},
		{"empty bearer", "Bearer "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := runAuthMiddleware(t, tc.header)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401, got %d", w.Code)
			}
		})
	}
}

func TestAuthMiddleware_InvalidToken_Returns401(t *testing.T) {
	w := runAuthMiddleware(t, "Bearer this.is.not.a.valid.jwt")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	var body map[string]map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}
	if body["error"]["code"] != "INVALID_TOKEN" {
		t.Errorf("unexpected error code: %s", body["error"]["code"])
	}
}

func TestAuthMiddleware_ExpiredToken_Returns401(t *testing.T) {
	claims := jwt.MapClaims{
		"sub":  uuid.New().String(),
		"type": "access",
		"exp":  time.Now().Add(-1 * time.Hour).Unix(),
		"iat":  time.Now().Add(-2 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("failed to sign expired token: %v", err)
	}

	w := runAuthMiddleware(t, "Bearer "+signed)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired token, got %d", w.Code)
	}
}

func TestAuthMiddleware_WrongTokenType_Returns401(t *testing.T) {
	claims := jwt.MapClaims{
		"sub":  uuid.New().String(),
		"type": "refresh",
		"exp":  time.Now().Add(15 * time.Minute).Unix(),
		"iat":  time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	w := runAuthMiddleware(t, "Bearer "+signed)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong token type, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidTokenFormat_DecodesCorrectly(t *testing.T) {
	userID := uuid.New()
	tokenSvc := services.NewTokenService(testSettings())

	tokenStr := issueTestAccessToken(t, userID)
	payload, err := tokenSvc.Decode(tokenStr, "access")
	if err != nil {
		t.Fatalf("expected valid token to decode without error, got: %v", err)
	}
	if payload.Subject != userID {
		t.Errorf("expected subject %s, got %s", userID, payload.Subject)
	}
	if payload.Type != "access" {
		t.Errorf("expected type 'access', got %s", payload.Type)
	}
}
