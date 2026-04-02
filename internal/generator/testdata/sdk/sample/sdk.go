package sample

import (
	"context"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
)

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

type CreateWidgetResponse struct {
	Widget
}

type GetWidgetResponse struct {
	Widget
}

type ListWidgetsResponse struct {
	Items []WidgetSummary
}

type UpdateWidgetResponse struct {
	Widget
}

type DeleteWidgetResponse struct{}

type CreateOAuth2ClientCredentialDetails struct {
	Name string `mandatory:"true" json:"name"`

	Description string `mandatory:"false" json:"description,omitempty"`

	Scopes []string `mandatory:"true" json:"scopes"`
}

type UpdateOAuth2ClientCredentialDetails struct {
	Description string `mandatory:"true" json:"description"`

	Scopes []string `mandatory:"true" json:"scopes"`
}

type CreateOAuthClientCredentialRequest struct {
	CreateOAuth2ClientCredentialDetails `contributesTo:"body"`
}

type GetOAuthClientCredentialRequest struct{}
type ListOAuthClientCredentialsRequest struct{}

type UpdateOAuthClientCredentialRequest struct {
	UpdateOAuth2ClientCredentialDetails `contributesTo:"body"`
}

type DeleteOAuthClientCredentialRequest struct{}

type CreateOAuthClientCredentialResponse struct {
	OAuth2ClientCredential
}

type GetOAuthClientCredentialResponse struct {
	OAuth2ClientCredential
}

type ListOAuthClientCredentialsResponse struct {
	Items []OAuth2ClientCredential
}

type UpdateOAuthClientCredentialResponse struct {
	OAuth2ClientCredential
}

type DeleteOAuthClientCredentialResponse struct{}

type OAuth2ClientCredential struct {
	LifecycleState ModeEnum `mandatory:"false" json:"lifecycleState,omitempty"`
}

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

type GetReportResponse struct {
	Report
}

type ListReportsResponse struct {
	Items []ReportSummary
}

type DeleteReportResponse struct{}

type GetReportByNameDetails struct {
	DisplayName string `mandatory:"true" json:"displayName"`
}

type GetReportByNameRequest struct {
	GetReportByNameDetails `contributesTo:"body"`
}

type GetReportByNameResponse struct {
	Report
}

type CreateDbSystemDetails struct {
	CompartmentId string `json:"compartmentId,omitempty"`
	DisplayName   string `json:"displayName,omitempty"`
	Port          int    `json:"port,omitempty"`
}

type CreateDbSystemRequest struct{}
type GetDbSystemRequest struct{}
type DeleteDbSystemRequest struct{}

type CreateDbSystemResponse struct {
	DbSystem
}

type GetDbSystemResponse struct {
	DbSystem
}

type DeleteDbSystemResponse struct{}

type DbSystem struct {
	Id             string `json:"id,omitempty"`
	DisplayName    string `json:"displayName,omitempty"`
	CompartmentId  string `json:"compartmentId,omitempty"`
	Port           int    `json:"port,omitempty"`
	LifecycleState string `json:"lifecycleState,omitempty"`
}

type SampleClient struct{}

func NewSampleClientWithConfigurationProvider(common.ConfigurationProvider) (SampleClient, error) {
	return SampleClient{}, nil
}

func (SampleClient) CreateWidget(context.Context, CreateWidgetRequest) (CreateWidgetResponse, error) {
	return CreateWidgetResponse{}, nil
}

func (SampleClient) GetWidget(context.Context, GetWidgetRequest) (GetWidgetResponse, error) {
	return GetWidgetResponse{}, nil
}

func (SampleClient) ListWidgets(context.Context, ListWidgetsRequest) (ListWidgetsResponse, error) {
	return ListWidgetsResponse{}, nil
}

func (SampleClient) UpdateWidget(context.Context, UpdateWidgetRequest) (UpdateWidgetResponse, error) {
	return UpdateWidgetResponse{}, nil
}

func (SampleClient) DeleteWidget(context.Context, DeleteWidgetRequest) (DeleteWidgetResponse, error) {
	return DeleteWidgetResponse{}, nil
}

func (SampleClient) CreateOAuthClientCredential(context.Context, CreateOAuthClientCredentialRequest) (CreateOAuthClientCredentialResponse, error) {
	return CreateOAuthClientCredentialResponse{}, nil
}

func (SampleClient) GetOAuthClientCredential(context.Context, GetOAuthClientCredentialRequest) (GetOAuthClientCredentialResponse, error) {
	return GetOAuthClientCredentialResponse{}, nil
}

func (SampleClient) ListOAuthClientCredentials(context.Context, ListOAuthClientCredentialsRequest) (ListOAuthClientCredentialsResponse, error) {
	return ListOAuthClientCredentialsResponse{}, nil
}

func (SampleClient) UpdateOAuthClientCredential(context.Context, UpdateOAuthClientCredentialRequest) (UpdateOAuthClientCredentialResponse, error) {
	return UpdateOAuthClientCredentialResponse{}, nil
}

func (SampleClient) DeleteOAuthClientCredential(context.Context, DeleteOAuthClientCredentialRequest) (DeleteOAuthClientCredentialResponse, error) {
	return DeleteOAuthClientCredentialResponse{}, nil
}

func (SampleClient) GetReport(context.Context, GetReportRequest) (GetReportResponse, error) {
	return GetReportResponse{}, nil
}

func (SampleClient) ListReports(context.Context, ListReportsRequest) (ListReportsResponse, error) {
	return ListReportsResponse{}, nil
}

func (SampleClient) DeleteReport(context.Context, DeleteReportRequest) (DeleteReportResponse, error) {
	return DeleteReportResponse{}, nil
}

func (SampleClient) GetReportByName(context.Context, GetReportByNameRequest) (GetReportByNameResponse, error) {
	return GetReportByNameResponse{}, nil
}

func (SampleClient) CreateDbSystem(context.Context, CreateDbSystemRequest) (CreateDbSystemResponse, error) {
	return CreateDbSystemResponse{}, nil
}

func (SampleClient) GetDbSystem(context.Context, GetDbSystemRequest) (GetDbSystemResponse, error) {
	return GetDbSystemResponse{}, nil
}

func (SampleClient) DeleteDbSystem(context.Context, DeleteDbSystemRequest) (DeleteDbSystemResponse, error) {
	return DeleteDbSystemResponse{}, nil
}
