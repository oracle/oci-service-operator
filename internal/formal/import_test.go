package formal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const importTestManifest = "service\tslug\tkind\tstage\tsurface\timport\tspec\tlogic_gaps\tdiagram_dir\n" +
	"widget\twidget\tWidget\t%s\trepo-authored-semantics\timports/widget/widget.json\tcontrollers/widget/spec.cfg\tcontrollers/widget/logic-gaps.md\tcontrollers/widget/diagrams\n"

const importTestSpec = `# formal controller binding schema v1
schema_version = 1
surface = repo-authored-semantics
service = widget
slug = widget
kind = Widget
stage = %s
import = imports/widget/widget.json
shared_contracts = shared/BaseReconcilerContract.tla,shared/ControllerLifecycleSpec.tla,shared/OSOKServiceManagerContract.tla,shared/SecretSideEffectsContract.tla
status_projection = required
success_condition = active
requeue_conditions = provisioning,updating,terminating
delete_confirmation = required
finalizer_policy = retain-until-confirmed-delete
secret_side_effects = none
`

const importTestLogicGaps = `---
schemaVersion: 1
surface: repo-authored-semantics
service: widget
slug: widget
gaps:
  - category: seed-corpus
    status: open
    stopCondition: Replace with seeded semantics.
---

# Logic Gaps
`

const importTestDiagram = `schemaVersion: 1
surface: repo-authored-semantics
service: widget
slug: widget
kind: Widget
archetype: generated-service-manager
states:
  - provisioning
  - active
  - updating
  - terminating
notes:
  - Import test scaffold.
`

const importTestActivityPUML = `@startuml
title Activity - widget/Widget
start
:Load desired Widget state;
stop
@enduml
`

const importTestSequencePUML = `@startuml
title Sequence - widget/Widget
actor Kubernetes
participant Controller
Kubernetes -> Controller: reconcile
@enduml
`

const importTestStateMachinePUML = `@startuml
title State Machine - widget/Widget
[*] --> provisioning
provisioning --> active
active --> terminating
terminating --> [*]
@enduml
`

const importTestActivitySVG = `<svg xmlns="http://www.w3.org/2000/svg"><text>activity</text></svg>
`

const importTestSequenceSVG = `<svg xmlns="http://www.w3.org/2000/svg"><text>sequence</text></svg>
`

const importTestStateMachineSVG = `<svg xmlns="http://www.w3.org/2000/svg"><text>state-machine</text></svg>
`

const importTestImport = `{
  "schemaVersion": 1,
  "surface": "provider-facts",
  "service": "widget",
  "slug": "widget",
  "kind": "Widget",
  "sourceRef": "terraform-provider-oci",
  "providerResource": "%s",
  "operations": {
    "create": [
      {
        "operation": "CreateWidget",
        "requestType": "sdkwidget.CreateWidgetRequest",
        "responseType": "sdkwidget.CreateWidgetResponse"
      }
    ],
    "get": [
      {
        "operation": "GetWidget",
        "requestType": "sdkwidget.GetWidgetRequest",
        "responseType": "sdkwidget.GetWidgetResponse"
      }
    ],
    "list": [
      {
        "operation": "ListWidgets",
        "requestType": "sdkwidget.ListWidgetsRequest",
        "responseType": "sdkwidget.ListWidgetsResponse"
      }
    ],
    "update": [
      {
        "operation": "UpdateWidget",
        "requestType": "sdkwidget.UpdateWidgetRequest",
        "responseType": "sdkwidget.UpdateWidgetResponse"
      }
    ],
    "delete": [
      {
        "operation": "DeleteWidget",
        "requestType": "sdkwidget.DeleteWidgetRequest",
        "responseType": "sdkwidget.DeleteWidgetResponse"
      }
    ]
  },
  "lifecycle": {
    "create": {
      "pending": ["CREATING"],
      "target": ["ACTIVE"]
    },
    "update": {
      "pending": ["UPDATING"],
      "target": ["ACTIVE"]
    }
  },
  "mutation": {
    "mutable": ["display_name"],
    "forceNew": ["compartment_id"],
    "conflictsWith": {}
  },
  "hooks": {
    "create": [
      {
        "helper": "tfresource.CreateResource"
      }
    ],
    "update": [
      {
        "helper": "tfresource.UpdateResource"
      }
    ],
    "delete": [
      {
        "helper": "tfresource.DeleteResource"
      }
    ]
  },
  "deleteConfirmation": {
    "pending": ["DELETING"],
    "target": ["DELETED"]
  },
  "listLookup": {
    "datasource": "oci_widget_widgets",
    "collectionField": "widgets",
    "responseItemsField": "Items",
    "filterFields": ["compartment_id", "name", "state"]
  },
  "boundary": {
    "providerFactsOnly": true,
    "repoAuthoredSpecPath": "controllers/widget/spec.cfg",
    "repoAuthoredLogicGapsPath": "controllers/widget/logic-gaps.md",
    "excludedSemantics": [
      "secret-output",
      "delete-confirmation"
    ]
  }
}
`

func TestImportRefreshesProviderFactsAndPinsSource(t *testing.T) {
	formalRoot := writeFormalRoot(t, "seeded", "oci_widget_widget")
	providerRoot := writeProviderFixture(t)

	report, err := Import(ImportOptions{
		Root:             formalRoot,
		ProviderPath:     providerRoot,
		ProviderRevision: "test-revision",
	})
	if err != nil {
		t.Fatalf("Import() returned error: %v", err)
	}

	if report.ModulePath != "example.com/provider" {
		t.Fatalf("report.ModulePath = %q, want %q", report.ModulePath, "example.com/provider")
	}
	if report.Revision != "test-revision" {
		t.Fatalf("report.Revision = %q, want %q", report.Revision, "test-revision")
	}
	if report.ImportsRefreshed != 1 {
		t.Fatalf("report.ImportsRefreshed = %d, want 1", report.ImportsRefreshed)
	}
	if report.ScaffoldRowsSkipped != 0 {
		t.Fatalf("report.ScaffoldRowsSkipped = %d, want 0", report.ScaffoldRowsSkipped)
	}

	doc, err := loadImport(filepath.Join(formalRoot, "imports", "widget", "widget.json"))
	if err != nil {
		t.Fatalf("loadImport() failed: %v", err)
	}

	assertBindings(t, doc.Operations.Create, []operationBinding{{
		Operation:    "CreateWidget",
		RequestType:  "sdkwidget.CreateWidgetRequest",
		ResponseType: "sdkwidget.CreateWidgetResponse",
	}})
	assertBindings(t, doc.Operations.Get, []operationBinding{{
		Operation:    "GetWidget",
		RequestType:  "sdkwidget.GetWidgetRequest",
		ResponseType: "sdkwidget.GetWidgetResponse",
	}})
	assertBindings(t, doc.Operations.Update, []operationBinding{
		{
			Operation:    "ChangeWidgetCompartment",
			RequestType:  "sdkwidget.ChangeWidgetCompartmentRequest",
			ResponseType: "sdkwidget.ChangeWidgetCompartmentResponse",
		},
		{
			Operation:    "UpdateWidget",
			RequestType:  "sdkwidget.UpdateWidgetRequest",
			ResponseType: "sdkwidget.UpdateWidgetResponse",
		},
	})
	assertBindings(t, doc.Operations.Delete, []operationBinding{{
		Operation:    "DeleteWidget",
		RequestType:  "sdkwidget.DeleteWidgetRequest",
		ResponseType: "sdkwidget.DeleteWidgetResponse",
	}})
	assertBindings(t, doc.Operations.List, []operationBinding{{
		Operation:    "ListWidgets",
		RequestType:  "sdkwidget.ListWidgetsRequest",
		ResponseType: "sdkwidget.ListWidgetsResponse",
	}})

	assertStrings(t, doc.Lifecycle.Create.Pending, []string{"CREATING"})
	assertStrings(t, doc.Lifecycle.Create.Target, []string{"ACTIVE"})
	assertStrings(t, doc.Lifecycle.Update.Pending, []string{"UPDATING"})
	assertStrings(t, doc.Lifecycle.Update.Target, []string{"ACTIVE"})
	assertStrings(t, doc.DeleteConfirmation.Pending, []string{"DELETING"})
	assertStrings(t, doc.DeleteConfirmation.Target, []string{"DELETED"})

	assertStrings(t, doc.Mutation.Mutable, []string{"delete_associated", "display_name", "name", "pool_id", "settings.mode"})
	assertStrings(t, doc.Mutation.ForceNew, []string{"compartment_id", "settings.shape"})
	assertStrings(t, doc.Mutation.ConflictsWith["pool_id"], []string{"compartment_id"})

	assertHooks(t, doc.Hooks.Create, []hook{
		{Helper: "tfresource.CreateResource"},
		{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "widget", Action: "CREATED"},
	})
	assertHooks(t, doc.Hooks.Update, []hook{
		{Helper: "tfresource.UpdateResource"},
		{Helper: "tfresource.WaitForUpdatedState"},
		{Helper: "tfresource.WaitForWorkRequestWithErrorHandling", EntityType: "widget", Action: "UPDATED"},
	})
	assertHooks(t, doc.Hooks.Delete, []hook{
		{Helper: "tfresource.DeleteResource"},
	})

	if doc.ListLookup == nil {
		t.Fatal("doc.ListLookup = nil, want populated lookup")
	}
	if doc.ListLookup.Datasource != "oci_widget_widgets" {
		t.Fatalf("doc.ListLookup.Datasource = %q, want %q", doc.ListLookup.Datasource, "oci_widget_widgets")
	}
	if doc.ListLookup.CollectionField != "widgets" {
		t.Fatalf("doc.ListLookup.CollectionField = %q, want %q", doc.ListLookup.CollectionField, "widgets")
	}
	if doc.ListLookup.ResponseItemsField != "Items" {
		t.Fatalf("doc.ListLookup.ResponseItemsField = %q, want %q", doc.ListLookup.ResponseItemsField, "Items")
	}
	assertStrings(t, doc.ListLookup.FilterFields, []string{"compartment_id", "name", "state"})

	lockFile, _, err := loadSourceLock(filepath.Join(formalRoot, "sources.lock"))
	if err != nil {
		t.Fatalf("loadSourceLock() failed: %v", err)
	}
	if len(lockFile.Sources) != 1 {
		t.Fatalf("len(lockFile.Sources) = %d, want 1", len(lockFile.Sources))
	}
	if lockFile.Sources[0].Status != "pinned" {
		t.Fatalf("sources.lock status = %q, want %q", lockFile.Sources[0].Status, "pinned")
	}
	if lockFile.Sources[0].Path != "example.com/provider" {
		t.Fatalf("sources.lock path = %q, want %q", lockFile.Sources[0].Path, "example.com/provider")
	}
	if lockFile.Sources[0].Revision != "test-revision" {
		t.Fatalf("sources.lock revision = %q, want %q", lockFile.Sources[0].Revision, "test-revision")
	}

	if _, err := Verify(formalRoot); err != nil {
		t.Fatalf("Verify(%q) returned error after import: %v", formalRoot, err)
	}
}

func TestImportSkipsScaffoldRows(t *testing.T) {
	formalRoot := writeFormalRoot(t, "scaffold", "template_resource")
	providerRoot := writeProviderFixture(t)

	report, err := Import(ImportOptions{
		Root:             formalRoot,
		ProviderPath:     providerRoot,
		ProviderRevision: "test-revision",
	})
	if err != nil {
		t.Fatalf("Import() returned error: %v", err)
	}
	if report.ImportsRefreshed != 0 {
		t.Fatalf("report.ImportsRefreshed = %d, want 0", report.ImportsRefreshed)
	}
	if report.ScaffoldRowsSkipped != 1 {
		t.Fatalf("report.ScaffoldRowsSkipped = %d, want 1", report.ScaffoldRowsSkipped)
	}

	lockFile, _, err := loadSourceLock(filepath.Join(formalRoot, "sources.lock"))
	if err != nil {
		t.Fatalf("loadSourceLock() failed: %v", err)
	}
	if lockFile.Sources[0].Status != "pinned" {
		t.Fatalf("sources.lock status = %q, want %q", lockFile.Sources[0].Status, "pinned")
	}
}

func TestNormalizeResponseItemsField(t *testing.T) {
	if got := normalizeResponseItemsField(""); got != "Items" {
		t.Fatalf("normalizeResponseItemsField(\"\") = %q, want %q", got, "Items")
	}
	if got := normalizeResponseItemsField("Entries"); got != "Entries" {
		t.Fatalf("normalizeResponseItemsField(\"Entries\") = %q, want %q", got, "Entries")
	}
}

func writeFormalRoot(t *testing.T, stage, providerResource string) string {
	t.Helper()

	root := t.TempDir()
	writeImportTestFile(t, filepath.Join(root, "controller_manifest.tsv"), []byte(sprintf(importTestManifest, stage)))
	writeImportTestFile(t, filepath.Join(root, "sources.lock"), []byte(testSourcesLock))
	writeImportTestFile(t, filepath.Join(root, "shared", "BaseReconcilerContract.tla"), []byte(baseReconcilerContract))
	writeImportTestFile(t, filepath.Join(root, "shared", "ControllerLifecycleSpec.tla"), []byte(controllerLifecycleContract))
	writeImportTestFile(t, filepath.Join(root, "shared", "OSOKServiceManagerContract.tla"), []byte(serviceManagerContract))
	writeImportTestFile(t, filepath.Join(root, "shared", "SecretSideEffectsContract.tla"), []byte(secretSideEffectsContract))
	writeControllerDiagramStrategyFixtures(t, root)
	writeImportTestFile(t, filepath.Join(root, "controllers", "widget", "spec.cfg"), []byte(sprintf(importTestSpec, stage)))
	writeImportTestFile(t, filepath.Join(root, "controllers", "widget", "logic-gaps.md"), []byte(importTestLogicGaps))
	writeImportTestFile(t, filepath.Join(root, "controllers", "widget", "diagrams", "runtime-lifecycle.yaml"), []byte(importTestDiagram))
	writeImportTestFile(t, filepath.Join(root, "imports", "widget", "widget.json"), []byte(sprintf(importTestImport, providerResource)))
	requirePlantUML(t)
	if _, err := RenderDiagrams(RenderOptions{Root: root}); err != nil {
		t.Fatalf("RenderDiagrams(%q) error = %v", root, err)
	}
	return root
}

func writeProviderFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeImportTestFile(t, filepath.Join(root, "go.mod"), []byte("module example.com/provider\n\ngo 1.22\n"))
	writeImportTestFile(t, filepath.Join(root, "internal", "schema", "schema.go"), []byte(`package schema

import "time"

type ValueType int

const (
	TypeString ValueType = iota
	TypeList
	TypeBool
	TypeInt
)

const (
	TimeoutCreate = "create"
	TimeoutUpdate = "update"
	TimeoutDelete = "delete"
)

type Resource struct {
	Importer *ResourceImporter
	Timeouts *ResourceTimeout
	Create   func(*ResourceData, interface{}) error
	Read     func(*ResourceData, interface{}) error
	Update   func(*ResourceData, interface{}) error
	Delete   func(*ResourceData, interface{}) error
	Schema   map[string]*Schema
}

type ResourceImporter struct {
	State interface{}
}

var ImportStatePassthrough interface{}

type ResourceTimeout struct {
	Create *time.Duration
	Update *time.Duration
	Delete *time.Duration
}

type Schema struct {
	Type          ValueType
	Required      bool
	Optional      bool
	Computed      bool
	ForceNew      bool
	ConflictsWith []string
	Elem          interface{}
}

type ResourceData struct{}

func (d *ResourceData) GetOkExists(string) (interface{}, bool) { return nil, false }
func (d *ResourceData) HasChange(string) bool { return false }
func (d *ResourceData) GetChange(string) (interface{}, interface{}) { return nil, nil }
func (d *ResourceData) Timeout(string) time.Duration { return 0 }
func (d *ResourceData) SetId(string) {}
func (d *ResourceData) Id() string { return "" }
`))
	writeImportTestFile(t, filepath.Join(root, "internal", "tfresource", "tfresource.go"), []byte(`package tfresource

import (
	"time"

	"example.com/provider/internal/schema"
)

var DefaultTimeout = &schema.ResourceTimeout{}

func RegisterResource(string, *schema.Resource) {}
func RegisterDatasource(string, *schema.Resource) {}
func CreateResource(*schema.ResourceData, interface{}) error { return nil }
func ReadResource(interface{}) error { return nil }
func UpdateResource(*schema.ResourceData, interface{}) error { return nil }
func DeleteResource(*schema.ResourceData, interface{}, ...error) error { return nil }
func WaitForUpdatedState(*schema.ResourceData, interface{}) error { return nil }
func WaitForWorkRequestWithErrorHandling(interface{}, *string, string, interface{}, time.Duration, bool) (*string, error) {
	return nil, nil
}
func GetDataSourceItemSchema(*schema.Resource) *schema.Resource { return nil }
`))
	writeImportTestFile(t, filepath.Join(root, "internal", "client", "client.go"), []byte(`package client

import "example.com/provider/internal/sdkwidget"

type OracleClients struct{}

func (c *OracleClients) WidgetClient() *sdkwidget.WidgetClient {
	return &sdkwidget.WidgetClient{}
}
`))
	writeImportTestFile(t, filepath.Join(root, "internal", "sdkwidget", "sdkwidget.go"), []byte(`package sdkwidget

import "context"

type WidgetLifecycleState string

const (
	WidgetLifecycleStateCreating WidgetLifecycleState = "CREATING"
	WidgetLifecycleStateActive   WidgetLifecycleState = "ACTIVE"
	WidgetLifecycleStateUpdating WidgetLifecycleState = "UPDATING"
	WidgetLifecycleStateDeleting WidgetLifecycleState = "DELETING"
	WidgetLifecycleStateDeleted  WidgetLifecycleState = "DELETED"
)

type WorkRequestResourceActionType string

const (
	WorkRequestResourceActionTypeCreated WorkRequestResourceActionType = "CREATED"
	WorkRequestResourceActionTypeUpdated WorkRequestResourceActionType = "UPDATED"
)

type RequestMetadata struct{}

type Widget struct {
	Id *string
}

type WidgetSummary struct{}

type CreateWidgetRequest struct {
	Name            *string
	RequestMetadata RequestMetadata
}

type CreateWidgetResponse struct {
	Widget           Widget
	OpcWorkRequestId *string
}

type GetWidgetRequest struct {
	WidgetId         *string
	RequestMetadata  RequestMetadata
}

type GetWidgetResponse struct {
	Widget Widget
}

type UpdateWidgetRequest struct {
	WidgetId        *string
	DisplayName     *string
	RequestMetadata RequestMetadata
}

type UpdateWidgetResponse struct {
	Widget           Widget
	OpcWorkRequestId *string
}

type ChangeWidgetCompartmentRequest struct {
	WidgetId        *string
	CompartmentId   *string
	RequestMetadata RequestMetadata
}

type ChangeWidgetCompartmentResponse struct {
	Widget           Widget
	OpcWorkRequestId *string
}

type DeleteWidgetRequest struct {
	WidgetId         *string
	DeleteAssociated *bool
	RequestMetadata  RequestMetadata
}

type DeleteWidgetResponse struct{}

type ListWidgetsRequest struct {
	CompartmentId    *string
	Name             *string
	State            *string
	Page             *string
	RequestMetadata  RequestMetadata
}

type ListWidgetsResponse struct {
	Items       []WidgetSummary
	OpcNextPage *string
}

type WidgetClient struct{}

func (c *WidgetClient) CreateWidget(context.Context, CreateWidgetRequest) (CreateWidgetResponse, error) {
	return CreateWidgetResponse{}, nil
}

func (c *WidgetClient) GetWidget(context.Context, GetWidgetRequest) (GetWidgetResponse, error) {
	return GetWidgetResponse{}, nil
}

func (c *WidgetClient) UpdateWidget(context.Context, UpdateWidgetRequest) (UpdateWidgetResponse, error) {
	return UpdateWidgetResponse{}, nil
}

func (c *WidgetClient) ChangeWidgetCompartment(context.Context, ChangeWidgetCompartmentRequest) (ChangeWidgetCompartmentResponse, error) {
	return ChangeWidgetCompartmentResponse{}, nil
}

func (c *WidgetClient) DeleteWidget(context.Context, DeleteWidgetRequest) (DeleteWidgetResponse, error) {
	return DeleteWidgetResponse{}, nil
}

func (c *WidgetClient) ListWidgets(context.Context, ListWidgetsRequest) (ListWidgetsResponse, error) {
	return ListWidgetsResponse{}, nil
}
`))
	writeImportTestFile(t, filepath.Join(root, "internal", "service", "widget", "register_resource.go"), []byte(`package widget

import "example.com/provider/internal/tfresource"

func RegisterResource() {
	tfresource.RegisterResource("oci_widget_widget", WidgetResource())
}
`))
	writeImportTestFile(t, filepath.Join(root, "internal", "service", "widget", "register_datasource.go"), []byte(`package widget

import "example.com/provider/internal/tfresource"

func RegisterDatasource() {
	tfresource.RegisterDatasource("oci_widget_widgets", WidgetsDataSource())
}
`))
	writeImportTestFile(t, filepath.Join(root, "internal", "service", "widget", "widget_resource.go"), []byte(`package widget

import (
	"context"

	"example.com/provider/internal/client"
	"example.com/provider/internal/schema"
	"example.com/provider/internal/sdkwidget"
	"example.com/provider/internal/tfresource"
)

func WidgetResource() *schema.Resource {
	return &schema.Resource{
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Timeouts: tfresource.DefaultTimeout,
		Create:   createWidget,
		Read:     readWidget,
		Update:   updateWidget,
		Delete:   deleteWidget,
		Schema: map[string]*schema.Schema{
			"compartment_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"display_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"pool_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"compartment_id"},
			},
			"settings": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mode": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"shape": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},
			"delete_associated": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func createWidget(d *schema.ResourceData, m interface{}) error {
	sync := &WidgetResourceCrud{}
	sync.D = d
	sync.Client = m.(*client.OracleClients).WidgetClient()
	return tfresource.CreateResource(d, sync)
}

func readWidget(d *schema.ResourceData, m interface{}) error {
	sync := &WidgetResourceCrud{}
	sync.D = d
	sync.Client = m.(*client.OracleClients).WidgetClient()
	return tfresource.ReadResource(sync)
}

func updateWidget(d *schema.ResourceData, m interface{}) error {
	sync := &WidgetResourceCrud{}
	sync.D = d
	sync.Client = m.(*client.OracleClients).WidgetClient()
	return tfresource.UpdateResource(d, sync)
}

func deleteWidget(d *schema.ResourceData, m interface{}) error {
	sync := &WidgetResourceCrud{}
	sync.D = d
	sync.Client = m.(*client.OracleClients).WidgetClient()
	return tfresource.DeleteResource(d, sync)
}

type WidgetResourceCrud struct {
	D                 *schema.ResourceData
	Client            *sdkwidget.WidgetClient
	Res               *sdkwidget.Widget
	WorkRequestClient interface{}
}

func (s *WidgetResourceCrud) ID() string { return "" }
func (s *WidgetResourceCrud) VoidState() {}
func (s *WidgetResourceCrud) SetData() error { return nil }

func (s *WidgetResourceCrud) CreatedPending() []string {
	return []string{string(sdkwidget.WidgetLifecycleStateCreating)}
}

func (s *WidgetResourceCrud) CreatedTarget() []string {
	return []string{string(sdkwidget.WidgetLifecycleStateActive)}
}

func (s *WidgetResourceCrud) UpdatedPending() []string {
	return []string{string(sdkwidget.WidgetLifecycleStateUpdating)}
}

func (s *WidgetResourceCrud) UpdatedTarget() []string {
	return []string{string(sdkwidget.WidgetLifecycleStateActive)}
}

func (s *WidgetResourceCrud) DeletedPending() []string {
	return []string{string(sdkwidget.WidgetLifecycleStateDeleting)}
}

func (s *WidgetResourceCrud) DeletedTarget() []string {
	return []string{string(sdkwidget.WidgetLifecycleStateDeleted)}
}

func (s *WidgetResourceCrud) Create() error {
	request := sdkwidget.CreateWidgetRequest{}
	if name, ok := s.D.GetOkExists("name"); ok {
		tmp := name.(string)
		request.Name = &tmp
	}
	response, err := s.Client.CreateWidget(context.Background(), request)
	if err != nil {
		return err
	}
	_, err = tfresource.WaitForWorkRequestWithErrorHandling(s.WorkRequestClient, response.OpcWorkRequestId, "widget", sdkwidget.WorkRequestResourceActionTypeCreated, s.D.Timeout(schema.TimeoutCreate), false)
	if err != nil {
		return err
	}
	s.Res = &response.Widget
	return nil
}

func (s *WidgetResourceCrud) Get() error {
	request := sdkwidget.GetWidgetRequest{}
	tmp := s.D.Id()
	request.WidgetId = &tmp
	response, err := s.Client.GetWidget(context.Background(), request)
	if err != nil {
		return err
	}
	s.Res = &response.Widget
	return nil
}

func (s *WidgetResourceCrud) Update() error {
	if compartmentId, ok := s.D.GetOkExists("compartment_id"); ok && s.D.HasChange("compartment_id") {
		if err := s.updateCompartment(compartmentId); err != nil {
			return err
		}
	}
	request := sdkwidget.UpdateWidgetRequest{}
	tmp := s.D.Id()
	request.WidgetId = &tmp
	if displayName, ok := s.D.GetOkExists("display_name"); ok && s.D.HasChange("display_name") {
		tmp := displayName.(string)
		request.DisplayName = &tmp
	}
	response, err := s.Client.UpdateWidget(context.Background(), request)
	if err != nil {
		return err
	}
	s.Res = &response.Widget
	return tfresource.WaitForUpdatedState(s.D, s)
}

func (s *WidgetResourceCrud) updateCompartment(compartmentId interface{}) error {
	request := sdkwidget.ChangeWidgetCompartmentRequest{}
	tmp := s.D.Id()
	request.WidgetId = &tmp
	value := compartmentId.(string)
	request.CompartmentId = &value
	response, err := s.Client.ChangeWidgetCompartment(context.Background(), request)
	if err != nil {
		return err
	}
	_, err = tfresource.WaitForWorkRequestWithErrorHandling(s.WorkRequestClient, response.OpcWorkRequestId, "widget", sdkwidget.WorkRequestResourceActionTypeUpdated, s.D.Timeout(schema.TimeoutUpdate), false)
	return err
}

func (s *WidgetResourceCrud) Delete() error {
	request := sdkwidget.DeleteWidgetRequest{}
	tmp := s.D.Id()
	request.WidgetId = &tmp
	if deleteAssociated, ok := s.D.GetOkExists("delete_associated"); ok {
		tmp := deleteAssociated.(bool)
		request.DeleteAssociated = &tmp
	}
	_, err := s.Client.DeleteWidget(context.Background(), request)
	return err
}
`))
	writeImportTestFile(t, filepath.Join(root, "internal", "service", "widget", "widgets_data_source.go"), []byte(`package widget

import (
	"context"

	"example.com/provider/internal/client"
	"example.com/provider/internal/schema"
	"example.com/provider/internal/sdkwidget"
	"example.com/provider/internal/tfresource"
)

func WidgetsDataSource() *schema.Resource {
	return &schema.Resource{
		Read: readWidgets,
		Schema: map[string]*schema.Schema{
			"filter": {
				Type:     schema.TypeList,
				Optional: true,
			},
			"compartment_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"state": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"widgets": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     tfresource.GetDataSourceItemSchema(WidgetResource()),
			},
		},
	}
}

func readWidgets(d *schema.ResourceData, m interface{}) error {
	sync := &WidgetsDataSourceCrud{}
	sync.D = d
	sync.Client = m.(*client.OracleClients).WidgetClient()
	return tfresource.ReadResource(sync)
}

type WidgetsDataSourceCrud struct {
	D      *schema.ResourceData
	Client *sdkwidget.WidgetClient
	Res    *sdkwidget.ListWidgetsResponse
}

func (s *WidgetsDataSourceCrud) VoidState() {}

func (s *WidgetsDataSourceCrud) Get() error {
	request := sdkwidget.ListWidgetsRequest{}
	if compartmentId, ok := s.D.GetOkExists("compartment_id"); ok {
		tmp := compartmentId.(string)
		request.CompartmentId = &tmp
	}
	if name, ok := s.D.GetOkExists("name"); ok {
		tmp := name.(string)
		request.Name = &tmp
	}
	if state, ok := s.D.GetOkExists("state"); ok {
		tmp := state.(string)
		request.State = &tmp
	}
	response, err := s.Client.ListWidgets(context.Background(), request)
	if err != nil {
		return err
	}
	s.Res = &response
	request.Page = s.Res.OpcNextPage
	for request.Page != nil {
		listResponse, err := s.Client.ListWidgets(context.Background(), request)
		if err != nil {
			return err
		}
		s.Res.Items = append(s.Res.Items, listResponse.Items...)
		request.Page = listResponse.OpcNextPage
	}
	return nil
}

func (s *WidgetsDataSourceCrud) SetData() error {
	for _, item := range s.Res.Items {
		_ = item
	}
	return nil
}
`))
	return root
}

func writeImportTestFile(t *testing.T, path string, contents []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) failed: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) failed: %v", path, err)
	}
}

func assertBindings(t *testing.T, actual, want []operationBinding) {
	t.Helper()
	if len(actual) != len(want) {
		t.Fatalf("len(bindings) = %d, want %d: %#v", len(actual), len(want), actual)
	}
	for i := range want {
		if actual[i] != want[i] {
			t.Fatalf("binding[%d] = %#v, want %#v", i, actual[i], want[i])
		}
	}
}

func assertHooks(t *testing.T, actual, want []hook) {
	t.Helper()
	if len(actual) != len(want) {
		t.Fatalf("len(hooks) = %d, want %d: %#v", len(actual), len(want), actual)
	}
	for i := range want {
		if actual[i] != want[i] {
			t.Fatalf("hook[%d] = %#v, want %#v", i, actual[i], want[i])
		}
	}
}

func assertStrings(t *testing.T, actual, want []string) {
	t.Helper()
	if len(actual) != len(want) {
		t.Fatalf("len(strings) = %d, want %d: %#v", len(actual), len(want), actual)
	}
	for i := range want {
		if actual[i] != want[i] {
			t.Fatalf("strings[%d] = %q, want %q", i, actual[i], want[i])
		}
	}
}

func sprintf(format string, args ...any) string {
	return strings.TrimSuffix(fmt.Sprintf(format, args...), "\n") + "\n"
}
