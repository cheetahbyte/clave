package license

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
	Active         bool
	Features       []string
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

func (l *License) GetID() uuid.UUID       { return l.ID }
func (l *License) GetProductID() uuid.UUID { return l.ProductID }
func (l *License) GetFeatures() []string   { return l.Features }
func (l *License) GetExpiresAt() time.Time { return l.ExpiresAt }
func (l *License) IsActive() bool         { return l.Active }
