package domain

import (
	"time"

	"github.com/google/uuid"
)

type License struct {
	ID             uuid.UUID
	ProductID      uuid.UUID
	LookupDigest   []byte
	KeyPhc         string
	CustomerEmail  string
	MaxActivations int32
	IsActive       bool
	Features       []string
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

// this struct is a license that can be exposed via api (stripped of important fields)
type ExposableLicense struct {
	ID uuid.UUID
}
