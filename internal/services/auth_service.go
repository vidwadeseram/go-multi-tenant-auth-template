package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/apperrors"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/dto"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/mailer"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/models"
	"github.com/vidwadeseram/go-multi-tenant-auth-template/internal/repository"
)

type AuthService struct {
	userRepo      *repository.UserRepository
	tokenRepo     *repository.TokenRepository
	tenantService *TenantService
	tokenService  *TokenService
	mailer        *mailer.Mailer
	db            *gorm.DB
	logger        *slog.Logger
}

func NewAuthService(userRepo *repository.UserRepository, tokenRepo *repository.TokenRepository, tenantService *TenantService, tokenService *TokenService, mailer *mailer.Mailer, db *gorm.DB, logger *slog.Logger) *AuthService {
	return &AuthService{userRepo: userRepo, tokenRepo: tokenRepo, tenantService: tenantService, tokenService: tokenService, mailer: mailer, db: db, logger: logger}
}

func (s *AuthService) Register(ctx context.Context, payload dto.RegisterRequest) (*models.User, error) {
	if _, err := s.userRepo.GetByEmail(ctx, payload.Email); err == nil {
		return nil, apperrors.New(400, "EMAIL_ALREADY_EXISTS", "A user with this email already exists.")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:        strings.ToLower(strings.TrimSpace(payload.Email)),
		PasswordHash: string(hashedPassword),
		FirstName:    strings.TrimSpace(payload.FirstName),
		LastName:     strings.TrimSpace(payload.LastName),
		IsActive:     true,
		IsVerified:   false,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	verificationToken, err := s.tokenService.VerificationToken(user.ID, user.Email)
	if err == nil {
		_ = s.mailer.Send(user.Email, "Verify your account", fmt.Sprintf("Welcome %s, your verification token is: %s", user.FirstName, verificationToken))
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, payload dto.LoginRequest, tenantSlug string) (*dto.TokenResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, payload.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(401, "INVALID_CREDENTIALS", "Invalid email or password.")
		}
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(payload.Password)) != nil {
		return nil, apperrors.New(401, "INVALID_CREDENTIALS", "Invalid email or password.")
	}
	if !user.IsActive {
		return nil, apperrors.New(403, "USER_INACTIVE", "User account is inactive.")
	}

	var tenantID *string
	if strings.TrimSpace(tenantSlug) != "" {
		tenantCtx, err := s.tenantService.ResolveForUser(ctx, user.ID, tenantSlug)
		if err != nil {
			return nil, err
		}
		if tenantCtx.Tenant != nil {
			value := tenantCtx.Tenant.ID.String()
			tenantID = &value
		}
	}

	tokenResponse, rawRefreshToken, err := s.tokenService.IssueTokenPair(user.ID, tenantID)
	if err != nil {
		return nil, err
	}

	refreshRecord := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: s.tokenService.HashToken(rawRefreshToken),
		ExpiresAt: time.Now().UTC().Add(time.Duration(s.tokenService.settings.JWTRefreshExpireDays) * 24 * time.Hour),
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.tokenRepo.CreateWithTx(ctx, tx, refreshRecord)
	}); err != nil {
		return nil, err
	}

	return tokenResponse, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*dto.TokenResponse, error) {
	payload, err := s.tokenService.Decode(refreshToken, "refresh")
	if err != nil {
		return nil, err
	}

	tokenRecord, err := s.tokenRepo.FindActiveByHash(ctx, s.tokenService.HashToken(refreshToken), payload.Subject)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(401, "INVALID_REFRESH_TOKEN", "Refresh token is invalid or expired.")
		}
		return nil, err
	}
	if tokenRecord.ExpiresAt.Before(time.Now().UTC()) {
		return nil, apperrors.New(401, "INVALID_REFRESH_TOKEN", "Refresh token is invalid or expired.")
	}

	tokenResponse, rawRefreshToken, err := s.tokenService.IssueTokenPair(payload.Subject, payload.TenantID)
	if err != nil {
		return nil, err
	}
	newTokenRecord := &models.RefreshToken{
		UserID:    payload.Subject,
		TokenHash: s.tokenService.HashToken(rawRefreshToken),
		ExpiresAt: time.Now().UTC().Add(time.Duration(s.tokenService.settings.JWTRefreshExpireDays) * 24 * time.Hour),
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.tokenRepo.RevokeWithTx(ctx, tx, tokenRecord); err != nil {
			return err
		}
		return s.tokenRepo.CreateWithTx(ctx, tx, newTokenRecord)
	}); err != nil {
		return nil, err
	}

	return tokenResponse, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	payload, err := s.tokenService.Decode(refreshToken, "refresh")
	if err != nil {
		return err
	}
	tokenRecord, err := s.tokenRepo.FindActiveByHash(ctx, s.tokenService.HashToken(refreshToken), payload.Subject)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.New(401, "INVALID_REFRESH_TOKEN", "Refresh token is invalid.")
		}
		return err
	}
	return s.tokenRepo.Revoke(ctx, tokenRecord)
}

func (s *AuthService) Me(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	user, err := s.userRepo.GetActiveByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(401, "USER_NOT_FOUND", "Authenticated user was not found.")
		}
		s.logger.Error("failed to load user", "error", err, "user_id", userID)
		return nil, err
	}
	return user, nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	payload, err := s.tokenService.Decode(token, "verify")
	if err != nil {
		return err
	}
	user, err := s.userRepo.GetActiveByID(ctx, payload.Subject)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.New(404, "USER_NOT_FOUND", "User not found.")
		}
		return err
	}
	if user.IsVerified {
		return apperrors.New(400, "ALREADY_VERIFIED", "Email is already verified.")
	}
	user.IsVerified = true
	return s.userRepo.Update(ctx, user)
}

func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	resetToken, err := s.tokenService.VerificationToken(user.ID, user.Email)
	if err != nil {
		return err
	}
	_ = s.mailer.Send(user.Email, "Password Reset", fmt.Sprintf("Your password reset token is: %s", resetToken))
	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, token string, newPassword string) error {
	payload, err := s.tokenService.Decode(token, "verify")
	if err != nil {
		return err
	}
	user, err := s.userRepo.GetActiveByID(ctx, payload.Subject)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.New(404, "USER_NOT_FOUND", "User not found.")
		}
		return err
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hashedPassword)
	return s.userRepo.Update(ctx, user)
}
