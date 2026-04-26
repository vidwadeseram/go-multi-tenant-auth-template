package services

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/apperrors"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/config"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/dto"
)

type AccessClaims struct {
	TenantID  *string `json:"tenant_id,omitempty"`
	TokenType string  `json:"type"`
	jwt.RegisteredClaims
}

type TokenPayload struct {
	Subject  uuid.UUID
	TenantID *string
	Type     string
}

type TokenService struct {
	settings *config.Settings
}

func NewTokenService(settings *config.Settings) *TokenService {
	return &TokenService{settings: settings}
}

func (s *TokenService) IssueTokenPair(userID uuid.UUID, tenantID *string) (*dto.TokenResponse, string, error) {
	accessToken, err := s.createToken(userID, tenantID, "access", time.Duration(s.settings.JWTAccessExpireMinutes)*time.Minute)
	if err != nil {
		return nil, "", err
	}
	refreshToken, err := s.createToken(userID, tenantID, "refresh", time.Duration(s.settings.JWTRefreshExpireDays)*24*time.Hour)
	if err != nil {
		return nil, "", err
	}

	return &dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    s.settings.JWTAccessExpireMinutes * 60,
		TenantID:     tenantID,
	}, refreshToken, nil
}

func (s *TokenService) createToken(userID uuid.UUID, tenantID *string, tokenType string, ttl time.Duration) (string, error) {
	expiresAt := time.Now().UTC().Add(ttl)
	claims := AccessClaims{
		TenantID:  tenantID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.settings.JWTSecret))
}

func (s *TokenService) Decode(tokenString string, expectedType string) (*TokenPayload, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apperrors.New(401, "INVALID_TOKEN", "Token signing method is invalid.")
		}
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, apperrors.New(401, "INVALID_TOKEN", "Token signing method is invalid.")
		}
		return []byte(s.settings.JWTSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, apperrors.New(401, "INVALID_TOKEN", "Token is invalid or expired.")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, apperrors.New(401, "INVALID_TOKEN", "Token is invalid or expired.")
	}
	tokenType, _ := claims["type"].(string)
	if tokenType == "" {
		tokenType = "access"
	}
	if tokenType != expectedType {
		return nil, apperrors.New(401, "INVALID_TOKEN_TYPE", "Token type is invalid.")
	}
	subjectValue, _ := claims["sub"].(string)
	subject, err := uuid.Parse(subjectValue)
	if err != nil {
		return nil, apperrors.New(401, "INVALID_TOKEN", "Token subject is invalid.")
	}

	var tenantID *string
	if rawTenantID, ok := claims["tenant_id"].(string); ok && rawTenantID != "" {
		tenantID = &rawTenantID
	}

	return &TokenPayload{Subject: subject, TenantID: tenantID, Type: tokenType}, nil
}

func (s *TokenService) HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *TokenService) VerificationToken(userID uuid.UUID, email string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID.String(),
		"email": email,
		"type":  "verify",
		"exp":   time.Now().UTC().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.settings.JWTSecret))
}
