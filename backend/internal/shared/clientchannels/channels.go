package clientchannels

import (
	"context"

	"github.com/google/uuid"
)

type Channel struct {
	Name        string `json:"name"`
	IsDefault   bool   `json:"isDefault"`
	Description string `json:"description,omitempty"`
}

type Lister interface {
	AvailableChannels(ctx context.Context, productID uuid.UUID, features []string) ([]Channel, error)
}
