package audit

import "time"

type AuditLogItem struct {
	ID           string     `json:"id"`
	Action       string     `json:"action"`
	ResourceType string     `json:"resourceType"`
	ResourceID   *string    `json:"resourceId"`
	AdminEmail   *string    `json:"adminEmail"`
	IP           *string    `json:"ip"`
	UserAgent    *string    `json:"userAgent"`
	CreatedAt    *time.Time `json:"createdAt"`
}

type AuditLogListResponse struct {
	Items    []AuditLogItem `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}
