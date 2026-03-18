package sample

import "time"

type ModeEnum string

type CreateWidgetDetails struct {
	// The OCID of the widget compartment.
	CompartmentId string `mandatory:"true" json:"compartmentId"`

	// The display name of the widget.
	DisplayName string `mandatory:"true" json:"displayName"`

	// The widget name.
	Name string `mandatory:"false" json:"name,omitempty"`

	// The number of widget instances.
	Count int `mandatory:"false" json:"count,omitempty"`

	// Whether the widget is enabled.
	Enabled bool `mandatory:"false" json:"enabled,omitempty"`

	// Additional labels for the widget.
	Labels map[string]string `mandatory:"false" json:"labels,omitempty"`

	// The supported widget mode.
	Mode ModeEnum `mandatory:"false" json:"mode,omitempty"`

	// When the widget was created.
	CreatedAt time.Time `mandatory:"false" json:"createdAt,omitempty"`

	// The server-generated widget state. Read-only.
	ServerState string `mandatory:"true" json:"serverState"`

	// Widget source details.
	Source WidgetSource `mandatory:"false" json:"source,omitempty"`
}

type UpdateWidgetDetails struct {
	// The updated widget name.
	Name string `mandatory:"false" json:"name,omitempty"`
}

type Widget struct {
	// The lifecycle state of the widget.
	LifecycleState ModeEnum `mandatory:"false" json:"lifecycleState,omitempty"`
}

type WidgetSummary struct {
	// The time the widget was last updated.
	TimeUpdated time.Time `mandatory:"false" json:"timeUpdated,omitempty"`
}

type WidgetSource struct {
	// The widget source type.
	Type string `mandatory:"false" json:"type,omitempty"`
}

type CreateWidgetRequest struct{}
type GetWidgetRequest struct{}
type ListWidgetsRequest struct{}
type UpdateWidgetRequest struct{}
type DeleteWidgetRequest struct{}

type Report struct {
	Id             string    `json:"id,omitempty"`
	LifecycleState ModeEnum  `json:"lifecycleState,omitempty"`
	TimeCreated    time.Time `json:"timeCreated,omitempty"`
}

type ReportSummary struct {
	DisplayName string `json:"displayName,omitempty"`
}

type GetReportRequest struct{}
type ListReportsRequest struct{}
type DeleteReportRequest struct{}

type CreateDbSystemDetails struct {
	CompartmentId string `json:"compartmentId,omitempty"`
	DisplayName   string `json:"displayName,omitempty"`
	Port          int    `json:"port,omitempty"`
}

type CreateDbSystemRequest struct{}
type GetDbSystemRequest struct{}
type DeleteDbSystemRequest struct{}
