package handlers

import "github.com/cheetahbyte/clave/internal/services"

type Handlers struct {
	Services services.ServiceStack
}

func New(s services.ServiceStack) *Handlers {
	return &Handlers{Services: s}
}
