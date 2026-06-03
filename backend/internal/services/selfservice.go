package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/handlers/dto"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidEmail = errors.New("invalid email")
)

type SelfServiceService struct {
	repo           *db.Queries
	tokenPepper    []byte
	signingService *SigningService
}

func NewSelfServiceService(
	repo *db.Queries,
	tokenPepper []byte,
	signingService *SigningService,
) *SelfServiceService {
	return &SelfServiceService{
		repo:           repo,
		tokenPepper:    tokenPepper,
		signingService: signingService,
	}
}

func (svc *SelfServiceService) RequestToken(ctx context.Context, data db.CreateSelfServiceLinkParams) (string, error) {
	email := strings.ToLower(strings.TrimSpace(data.Email))
	if email == "" || !strings.Contains(email, "@") {
		return "", ErrInvalidEmail
	}
	data.Email = email

	rawToken, err := generateURLToken(32)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	data.TokenHash = svc.hashToken(rawToken)

	_, err = svc.repo.CreateSelfServiceLink(ctx, data)
	if err != nil {
		return "", fmt.Errorf("create self service token: %w", err)
	}

	return rawToken, nil
}

func (svc *SelfServiceService) ValidateToken(
	ctx context.Context,
	data dto.SelfServiceValidateTokenRequest,
) (dto.SelfServiceValidateTokenResponse, error) {

	raw := strings.TrimSpace(data.Token)
	if raw == "" {
		return dto.SelfServiceValidateTokenResponse{}, errors.New("token not found")
	}

	tokenHash := svc.hashToken(raw)

	email, err := svc.repo.ConsumeSelfServiceToken(ctx, tokenHash)
	if err != nil {
		slog.Warn("self-service token consume failed", "err", err)
		return dto.SelfServiceValidateTokenResponse{}, errors.New("problem with token")
	}

	now := time.Now()
	expiresAt := now.Add(time.Hour)

	claims := jwt.MapClaims{
		"aud":   "selfservice",
		"sub":   email,
		"email": email,
		"iat":   now.Unix(),
		"exp":   expiresAt.Unix(),
	}

	jwtoken, err := svc.signingService.IssueAndSignSelfServiceToken(claims)
	if err != nil {
		return dto.SelfServiceValidateTokenResponse{}, err
	}

	return dto.SelfServiceValidateTokenResponse{
		Token:     jwtoken,
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

func (svc *SelfServiceService) hashToken(raw string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(raw))
	_, _ = h.Write(svc.tokenPepper)
	return hex.EncodeToString(h.Sum(nil))
}

func generateURLToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func parseIP(ip string) net.IP {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return nil
	}
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	return net.ParseIP(ip)
}
