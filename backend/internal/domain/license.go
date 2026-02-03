package domain

import "time"

type License struct {
	ID             int32
	ProductID      int32
	LookupDigest   []byte
	KeyPhc         string
	CustomerEmail  string
	MaxActivations int32
	IsActive       bool
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

// this struct is a license that can be exposed via api (stripped of important fields)
type ExposableLicense struct {
	ID int32
}
