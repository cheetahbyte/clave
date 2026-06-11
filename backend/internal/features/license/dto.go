package license

type CreationRequest struct {
	ProductID      string `json:"productId" validate:"required,uuid"`
	MaxActivations int32  `json:"maxActivations" validate:"required,gte=1"`
	CustomerEmail  string `json:"customerEmail" validate:"required,email"`
}

type CreationResponse struct {
	LicenseKey string `json:"licenseKey"`
}
