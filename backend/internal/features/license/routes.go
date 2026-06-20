package license

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterAdminRoutes(r chi.Router) {
	r.Get("/overview", h.AdminOverview)
	r.Get("/stats/timeseries", h.AdminTimeseries)
	r.Get("/trials", h.AdminListTrials)
	r.Get("/licenses", h.AdminListLicenses)
	r.Get("/licenses/{id}", h.AdminGetLicense)
	r.Post("/licenses", h.Create)
	r.Patch("/licenses/{id}", h.AdminUpdateLicense)
	r.Delete("/licenses/{id}", h.AdminDeleteLicense)
	r.Get("/devices", h.AdminListDevices)
	r.Delete("/devices/{deviceId}", h.AdminDeleteDevice)
	r.Get("/products", h.AdminListProducts)
	r.Post("/products", h.AdminCreateProduct)
	r.Get("/products/{id}", h.AdminGetProduct)
	r.Patch("/products/{id}", h.AdminUpdateProduct)
	r.Delete("/products/{id}", h.AdminDeleteProduct)

	// Product feature catalog
	r.Get("/products/{id}/features", h.AdminListProductFeatures)
	r.Post("/products/{id}/features", h.AdminCreateProductFeature)
	r.Patch("/products/{id}/features/{featureId}", h.AdminUpdateProductFeature)
	r.Delete("/products/{id}/features/{featureId}", h.AdminDeleteProductFeature)

	// Scheduled feature windows
	r.Get("/products/{id}/feature-windows", h.AdminListFeatureWindows)
	r.Post("/products/{id}/feature-windows", h.AdminCreateFeatureWindow)
	r.Patch("/products/{id}/feature-windows/{windowId}", h.AdminUpdateFeatureWindow)
	r.Delete("/products/{id}/feature-windows/{windowId}", h.AdminDeleteFeatureWindow)
}
