package selfservice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/shared/signing"
	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidEmail = errors.New("invalid email")

type Service struct {
	q      *db.Queries
	pepper []byte
	signer *signing.Service
}

func NewService(q *db.Queries, pepper []byte, signer *signing.Service) *Service {
	return &Service{
		q:      q,
		pepper: pepper,
		signer: signer,
	}
}

func (svc *Service) RequestToken(ctx context.Context, data db.CreateSelfServiceLinkParams) (string, error) {
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

	_, err = svc.q.CreateSelfServiceLink(ctx, data)
	if err != nil {
		return "", fmt.Errorf("create self service token: %w", err)
	}

	return rawToken, nil
}

func (svc *Service) ValidateToken(ctx context.Context, data ValidateTokenRequest) (ValidateTokenResponse, error) {
	raw := strings.TrimSpace(data.Token)
	if raw == "" {
		return ValidateTokenResponse{}, errors.New("token not found")
	}

	tokenHash := svc.hashToken(raw)

	email, err := svc.q.ConsumeSelfServiceToken(ctx, tokenHash)
	if err != nil {
		slog.Warn("self-service token consume failed", "err", err)
		return ValidateTokenResponse{}, errors.New("problem with token")
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

	jwtoken, err := svc.signer.IssueAndSignSelfServiceToken(claims)
	if err != nil {
		return ValidateTokenResponse{}, err
	}

	return ValidateTokenResponse{
		Token:     jwtoken,
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

func (svc *Service) hashToken(raw string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(raw))
	_, _ = h.Write(svc.pepper)
	return hex.EncodeToString(h.Sum(nil))
}

func generateURLToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
