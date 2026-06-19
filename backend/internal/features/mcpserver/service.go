package mcpserver

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const tokenPrefix = "clv_mcp_"

var ErrInvalidToken = errors.New("invalid mcp token")

type AuthInfo struct {
	OrganizationID uuid.UUID
	TokenID        string
}

type GeneratedToken struct {
	Token         string
	Prefix        string
	RegeneratedAt time.Time
}

type Service struct {
	repo *Repository
}

func NewService(q *db.Queries) *Service {
	return &Service{repo: NewRepository(q)}
}

func (s *Service) GetToken(ctx context.Context, orgID uuid.UUID) (*TokenResponse, error) {
	tok, err := s.repo.GetByOrganization(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			return &TokenResponse{Exists: false}, nil
		}
		return nil, err
	}

	resp := &TokenResponse{
		Exists: true,
		Prefix: tok.TokenPrefix,
	}
	if tok.RegeneratedAt.Valid {
		resp.RegeneratedAt = &tok.RegeneratedAt.Time
	}
	if tok.LastUsedAt.Valid {
		resp.LastUsedAt = &tok.LastUsedAt.Time
	}
	return resp, nil
}

func (s *Service) RegenerateToken(ctx context.Context, orgID, adminID uuid.UUID) (*GeneratedToken, error) {
	tokenID, err := randomHex(12)
	if err != nil {
		return nil, err
	}
	secret, err := randomSecret(32)
	if err != nil {
		return nil, err
	}

	raw := tokenPrefix + tokenID + "_" + secret
	hash := hashTokenSecret(tokenID, secret)
	prefix := raw
	if len(prefix) > 18 {
		prefix = prefix[:18] + "..."
	}

	row, err := s.repo.Upsert(ctx, orgID, tokenID, hash[:], prefix, adminID)
	if err != nil {
		return nil, err
	}

	return &GeneratedToken{
		Token:         raw,
		Prefix:        row.TokenPrefix,
		RegeneratedAt: row.RegeneratedAt.Time,
	}, nil
}

func (s *Service) VerifyBearerToken(ctx context.Context, raw string) (*AuthInfo, error) {
	tokenID, secret, ok := parseToken(raw)
	if !ok {
		return nil, ErrInvalidToken
	}

	row, err := s.repo.GetByTokenID(ctx, tokenID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	want := hashTokenSecret(tokenID, secret)
	if subtle.ConstantTimeCompare(row.TokenHash, want[:]) != 1 {
		return nil, ErrInvalidToken
	}

	_ = s.repo.TouchLastUsed(ctx, row.OrganizationID)
	return &AuthInfo{OrganizationID: row.OrganizationID, TokenID: tokenID}, nil
}

func parseToken(raw string) (string, string, bool) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, tokenPrefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(raw, tokenPrefix)
	parts := strings.SplitN(rest, "_", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func hashTokenSecret(tokenID, secret string) [32]byte {
	return sha256.Sum256([]byte(tokenID + "\x00" + secret))
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func randomSecret(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
