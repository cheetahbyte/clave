package services

import "github.com/cheetahbyte/clave/internal/db"

type ServiceStack struct {
	license *LicenseService
}

func InitServices(q *db.Queries) ServiceStack {
	return ServiceStack{license: NewLicenseService(q)}
}

func (s ServiceStack) License() *LicenseService { return s.license }
