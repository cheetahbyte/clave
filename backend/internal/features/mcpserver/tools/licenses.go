package tools

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cheetahbyte/clave/internal/features/license"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// orgIDFromRequest extracts the authenticated organization ID from the
// MCP request's bearer-token payload.
func orgIDFromRequest(req *mcp.CallToolRequest) (uuid.UUID, error) {
	if req.Extra == nil || req.Extra.TokenInfo == nil || req.Extra.TokenInfo.Extra == nil {
		return uuid.Nil, errors.New("no authenticated organization in request")
	}
	raw, ok := req.Extra.TokenInfo.Extra["organization_id"].(string)
	if !ok || raw == "" {
		return uuid.Nil, errors.New("missing organization_id in token")
	}
	return uuid.Parse(raw)
}

// ============ find-license ============

type FindLicenseInput struct {
	Query string `json:"query" jsonschema:"description=Email, license ID (UUID), or license key (LIC-XXXX-...)"`
}

type LicenseResult struct {
	ID              string     `json:"id"`
	CustomerEmail   string     `json:"customerEmail"`
	ProductName     string     `json:"productName"`
	ProductID       string     `json:"productId"`
	IsActive        bool       `json:"isActive"`
	IsTrial         bool       `json:"isTrial"`
	MaxActivations  int32      `json:"maxActivations"`
	ActivationCount int64      `json:"activationCount"`
	Features        []string   `json:"features"`
	CreatedAt       *time.Time `json:"createdAt,omitempty"`
	ExpiresAt       *time.Time `json:"expiresAt,omitempty"`
}

type FindLicenseOutput struct {
	Results []LicenseResult `json:"results"`
}

// LicenseTools holds the dependencies needed by the license MCP tools.
type LicenseTools struct {
	Svc *license.Service
}

func (t *LicenseTools) FindLicense(ctx context.Context, req *mcp.CallToolRequest, input FindLicenseInput) (*mcp.CallToolResult, FindLicenseOutput, error) {
	orgID, err := orgIDFromRequest(req)
	if err != nil {
		return nil, FindLicenseOutput{}, err
	}

	results, err := t.Svc.AdminFindLicense(ctx, orgID, input.Query)
	if err != nil {
		return nil, FindLicenseOutput{}, fmt.Errorf("lookup failed: %w", err)
	}
	if len(results) == 0 {
		return nil, FindLicenseOutput{Results: []LicenseResult{}}, nil
	}

	out := make([]LicenseResult, len(results))
	for i, r := range results {
		out[i] = LicenseResult{
			ID:              r.ID,
			CustomerEmail:   r.CustomerEmail,
			ProductName:     r.ProductName,
			ProductID:       r.ProductID,
			IsActive:        r.IsActive,
			IsTrial:         r.IsTrial,
			MaxActivations:  r.MaxActivations,
			ActivationCount: r.ActivationCount,
			Features:        r.Features,
			CreatedAt:       r.CreatedAt,
			ExpiresAt:       r.ExpiresAt,
		}
	}
	return nil, FindLicenseOutput{Results: out}, nil
}

// ============ create-license ============

type CreateLicenseInput struct {
	Email          string   `json:"email" jsonschema:"description=Customer email address"`
	ProductID      string   `json:"productId" jsonschema:"description=Product UUID"`
	Features       []string `json:"features,omitempty" jsonschema:"description=Optional feature keys to grant on the license"`
	MaxActivations int32    `json:"maxActivations,omitempty" jsonschema:"description=Maximum device activations (default 1)"`
}

type CreateLicenseOutput struct {
	LicenseKey  string `json:"licenseKey"`
	ProductName string `json:"productName"`
	IsTrial     bool   `json:"isTrial"`
}

func (t *LicenseTools) CreateLicense(ctx context.Context, req *mcp.CallToolRequest, input CreateLicenseInput) (*mcp.CallToolResult, CreateLicenseOutput, error) {
	orgID, err := orgIDFromRequest(req)
	if err != nil {
		return nil, CreateLicenseOutput{}, err
	}

	maxActivations := input.MaxActivations
	if maxActivations < 1 {
		maxActivations = 1
	}

	result, err := t.Svc.NewLicense(ctx, orgID, license.CreationRequest{
		ProductID:      input.ProductID,
		CustomerEmail:  input.Email,
		MaxActivations: maxActivations,
		Features:       input.Features,
	})
	if err != nil {
		return nil, CreateLicenseOutput{}, fmt.Errorf("failed to create license: %w", err)
	}

	return nil, CreateLicenseOutput{
		LicenseKey:  result.LicenseKey,
		ProductName: result.ProductName,
		IsTrial:     result.IsTrial,
	}, nil
}

// ============ revoke-license ============

type RevokeLicenseInput struct {
	Identifier string `json:"identifier" jsonschema:"description=License ID (UUID) or license key (LIC-XXXX-...)"`
}

type RevokeLicenseOutput struct {
	Revoked bool   `json:"revoked"`
	ID      string `json:"id,omitempty"`
}

func (t *LicenseTools) RevokeLicense(ctx context.Context, req *mcp.CallToolRequest, input RevokeLicenseInput) (*mcp.CallToolResult, RevokeLicenseOutput, error) {
	orgID, err := orgIDFromRequest(req)
	if err != nil {
		return nil, RevokeLicenseOutput{}, err
	}

	err = t.Svc.AdminRevokeLicense(ctx, orgID, input.Identifier)
	if err != nil {
		if errors.Is(err, license.ErrNotFound) {
			return nil, RevokeLicenseOutput{}, errors.New("license not found or already inactive")
		}
		return nil, RevokeLicenseOutput{}, fmt.Errorf("failed to revoke license: %w", err)
	}

	return nil, RevokeLicenseOutput{Revoked: true}, nil
}

// ============ list-licenses ============

type ListLicensesInput struct {
	Limit int32 `json:"limit,omitempty" jsonschema:"description=Number of recent licenses to return (default 10, max 20)"`
}

type ListLicensesOutput struct {
	Licenses []LicenseResult `json:"licenses"`
}

func (t *LicenseTools) ListLicenses(ctx context.Context, req *mcp.CallToolRequest, input ListLicensesInput) (*mcp.CallToolResult, ListLicensesOutput, error) {
	orgID, err := orgIDFromRequest(req)
	if err != nil {
		return nil, ListLicensesOutput{}, err
	}

	results, err := t.Svc.AdminListRecentLicenses(ctx, orgID, input.Limit)
	if err != nil {
		return nil, ListLicensesOutput{}, fmt.Errorf("failed to list licenses: %w", err)
	}

	out := make([]LicenseResult, len(results))
	for i, r := range results {
		out[i] = LicenseResult{
			ID:              r.ID,
			CustomerEmail:   r.CustomerEmail,
			ProductName:     r.ProductName,
			ProductID:       r.ProductID,
			IsActive:        r.IsActive,
			IsTrial:         r.IsTrial,
			MaxActivations:  r.MaxActivations,
			ActivationCount: r.ActivationCount,
			Features:        r.Features,
			CreatedAt:       r.CreatedAt,
			ExpiresAt:       r.ExpiresAt,
		}
	}
	return nil, ListLicensesOutput{Licenses: out}, nil
}
