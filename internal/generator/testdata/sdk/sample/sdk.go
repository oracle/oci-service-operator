package sample

import "time"

type ModeEnum string

type CreateWidgetDetails struct {
	CompartmentId string            `json:"compartmentId,omitempty"`
	DisplayName   string            `json:"displayName,omitempty"`
	Name          string            `json:"name,omitempty"`
	Count         int               `json:"count,omitempty"`
	Enabled       bool              `json:"enabled,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	Mode          ModeEnum          `json:"mode,omitempty"`
	CreatedAt     time.Time         `json:"createdAt,omitempty"`
	Source        WidgetSource      `json:"source,omitempty"`
}

type UpdateWidgetDetails struct {
	Name string `json:"name,omitempty"`
}

type Widget struct {
	LifecycleState ModeEnum `json:"lifecycleState,omitempty"`
}

type WidgetSource struct {
	Type string `json:"type,omitempty"`
}

type CreateWidgetRequest struct{}
type GetWidgetRequest struct{}
type ListWidgetsRequest struct{}
type UpdateWidgetRequest struct{}
type DeleteWidgetRequest struct{}

type CreateDbSystemDetails struct {
	CompartmentId string `json:"compartmentId,omitempty"`
	DisplayName   string `json:"displayName,omitempty"`
	Port          int    `json:"port,omitempty"`
}

type CreateDbSystemRequest struct{}
type GetDbSystemRequest struct{}
type DeleteDbSystemRequest struct{}
