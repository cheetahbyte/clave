package license

import "time"

type CreationRequest struct {
	ProductID      string `json:"productId" validate:"required,uuid"`
	MaxActivations int32  `json:"maxActivations" validate:"required,gte=1"`
	CustomerEmail  string `json:"customerEmail" validate:"required,email"`
	IsTrial        bool   `json:"isTrial"`
	TrialDays      int    `json:"trialDays" validate:"omitempty,gte=1,lte=365"`
	SendEmail      bool   `json:"sendEmail"`
}

type CreationResponse struct {
	LicenseKey  string `json:"licenseKey"`
	ProductName string `json:"productName"`
	IsTrial     bool   `json:"isTrial"`
}

type AdminOverview struct {
	TotalLicenses    int64              `json:"totalLicenses"`
	ActiveLicenses   int64              `json:"activeLicenses"`
	ExpiredLicenses  int64              `json:"expiredLicenses"`
	TotalProducts    int64              `json:"totalProducts"`
	TotalActivations int64              `json:"totalActivations"`
	TotalTrials      int64              `json:"totalTrials"`
	ActiveTrials     int64              `json:"activeTrials"`
	RecentLicenses   []AdminLicenseItem `json:"recentLicenses"`
}

type AdminTimeseriesPoint struct {
	Date        string `json:"date"`
	Licenses    int64  `json:"licenses"`
	Trials      int64  `json:"trials"`
	Activations int64  `json:"activations"`
}

type AdminLicenseItem struct {
	ID              string     `json:"id"`
	CustomerEmail   string     `json:"customerEmail"`
	ProductName     string     `json:"productName"`
	IsActive        bool       `json:"isActive"`
	IsTrial         bool       `json:"isTrial"`
	MaxActivations  int32      `json:"maxActivations"`
	ActivationCount int64      `json:"activationCount"`
	CreatedAt       *time.Time `json:"createdAt"`
	ExpiresAt       *time.Time `json:"expiresAt"`
}

type AdminLicenseListResponse struct {
	Items    []AdminLicenseItem `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
}

type AdminLicenseDetailResponse struct {
	ID              string                `json:"id"`
	CustomerEmail   string                `json:"customerEmail"`
	ProductName     string                `json:"productName"`
	ProductID       string                `json:"productId"`
	IsActive        bool                  `json:"isActive"`
	IsTrial         bool                  `json:"isTrial"`
	MaxActivations  int32                 `json:"maxActivations"`
	ActivationCount int64                 `json:"activationCount"`
	Features        []string              `json:"features"`
	CreatedAt       *time.Time            `json:"createdAt"`
	ExpiresAt       *time.Time            `json:"expiresAt"`
	Activations     []AdminActivationItem `json:"activations"`
}

type AdminActivationItem struct {
	ID          string     `json:"id"`
	DeviceID    string     `json:"deviceId"`
	Hostname    *string    `json:"hostname"`
	CreatedAt   *time.Time `json:"createdAt"`
	CheckedInAt *time.Time `json:"checkedInAt"`
}

type AdminProductItem struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Version   *string    `json:"version"`
	LogoURL   *string    `json:"logoUrl"`
	CreatedAt *time.Time `json:"createdAt"`
}

type CreateProductRequest struct {
	Name    string  `json:"name" validate:"required,min=1,max=200"`
	Version *string `json:"version"`
	LogoURL *string `json:"logoUrl"`
}

type UpdateProductRequest struct {
	Name    string  `json:"name" validate:"required,min=1,max=200"`
	Version *string `json:"version"`
	LogoURL *string `json:"logoUrl"`
}

type UpdateLicenseRequest struct {
	CustomerEmail  string     `json:"customerEmail" validate:"required,email"`
	MaxActivations int32      `json:"maxActivations" validate:"required,gte=1"`
	IsActive       bool       `json:"isActive"`
	ExpiresAt      *time.Time `json:"expiresAt"`
	Features       []string   `json:"features"`
}
