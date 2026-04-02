/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

// PackageModel is the intermediate representation rendered into generator-owned OSOK outputs.
type PackageModel struct {
	Service         ServiceConfig
	Domain          string
	Version         string
	GroupDNSName    string
	SampleOrder     int
	Resources       []ResourceModel
	Controller      ControllerOutputModel
	Registration    RegistrationOutputModel
	PackageOutput   PackageOutputModel
	ServiceManagers []ServiceManagerModel
}

// ControllerOutputModel describes the generated controller files owned by the generator contract.
type ControllerOutputModel struct {
	Resources []ControllerModel
}

// ControllerModel renders one controller source file under controllers/<group>/.
type ControllerModel struct {
	Kind                       string
	FileStem                   string
	ReconcilerType             string
	ResourceVariable           string
	MaxConcurrentReconciles    int
	HasMaxConcurrentReconciles bool
	RBACMarkers                []string
}

// RegistrationOutputModel describes one generated runtime registration file.
type RegistrationOutputModel struct {
	Group                 string
	APIImportPath         string
	APIImportAlias        string
	ControllerImportPath  string
	ControllerImportAlias string
	Resources             []RegistrationResourceModel
}

// RegistrationResourceModel renders one generated resource registration inside a group file.
type RegistrationResourceModel struct {
	Kind                      string
	ComponentName             string
	ReconcilerType            string
	ServiceManagerImportPath  string
	ServiceManagerImportAlias string
	WithDepsConstructor       string
}

// PackageOutputModel describes the non-API generated files owned by the generator contract.
type PackageOutputModel struct {
	Generate bool
	Metadata PackageMetadataModel
	Install  InstallKustomizationModel
}

// PackageMetadataModel renders to packages/<group>/metadata.env.
type PackageMetadataModel struct {
	PackageName            string
	PackageNamespace       string
	PackageNamePrefix      string
	CRDPaths               string
	RBACPaths              string
	DefaultControllerImage string
}

// InstallKustomizationModel renders to packages/<group>/install/kustomization.yaml.
type InstallKustomizationModel struct {
	Namespace   string
	NamePrefix  string
	Resources   []string
	PatchPath   string
	PatchTarget string
}

// ServiceManagerModel describes one generated pkg/servicemanager scaffold package.
type ServiceManagerModel struct {
	Kind                     string
	SDKName                  string
	FileStem                 string
	Formal                   *FormalModel
	Semantics                *RuntimeSemanticsModel
	PackagePath              string
	PackageName              string
	APIImportPath            string
	APIImportAlias           string
	SDKImportPath            string
	SDKImportAlias           string
	ManagerTypeName          string
	WithDepsConstructor      string
	Constructor              string
	ClientInterfaceName      string
	DefaultClientTypeName    string
	UsesCredentialClient     bool
	SDKClientTypeName        string
	SDKClientConstructor     string
	SDKClientConstructorKind string
	NeedsCredentialClient    bool
	CreateOperation          *RuntimeOperationModel
	GetOperation             *RuntimeOperationModel
	ListOperation            *RuntimeOperationModel
	UpdateOperation          *RuntimeOperationModel
	DeleteOperation          *RuntimeOperationModel
	ServiceClientFileName    string
	ServiceManagerFileName   string
}

// RuntimeOperationModel describes one generated SDK-backed operation binding.
type RuntimeOperationModel struct {
	MethodName       string
	RequestTypeName  string
	ResponseTypeName string
	UsesRequest      bool
	RequestFields    []RuntimeRequestFieldModel
}

// ResourceModel describes one generated top-level kind inside an OSOK API package.
type ResourceModel struct {
	SDKName             string
	Kind                string
	FileStem            string
	KindPlural          string
	Operations          []string
	Formal              *FormalModel
	Runtime             *RuntimeModel
	LeadingComments     []string
	SpecComments        []string
	HelperTypes         []TypeModel
	SpecFields          []FieldModel
	StatusTypeName      string
	StatusComments      []string
	StatusFields        []FieldModel
	PrintColumns        []PrintColumnModel
	ObjectComments      []string
	ListComments        []string
	Sample              SampleModel
	PrimaryDisplayField string
}

// RuntimeModel describes the OCI SDK client and methods that back a generated resource.
type RuntimeModel struct {
	ClientType            string
	ClientConstructor     string
	ClientConstructorKind string
	Semantics             *RuntimeSemanticsModel
	Create                *RuntimeOperationModel
	Get                   *RuntimeOperationModel
	List                  *RuntimeOperationModel
	Update                *RuntimeOperationModel
	Delete                *RuntimeOperationModel
}

// RuntimeRequestFieldModel describes one explicit generatedruntime request-field assignment.
type RuntimeRequestFieldModel struct {
	FieldName        string
	RequestName      string
	Contribution     string
	PreferResourceID bool
}

// RuntimeSemanticsModel describes the explicit formal runtime semantics attached to one resource.
type RuntimeSemanticsModel struct {
	FormalService       string
	FormalSlug          string
	StatusProjection    string
	SecretSideEffects   string
	FinalizerPolicy     string
	Lifecycle           RuntimeLifecycleModel
	Delete              RuntimeDeleteSemanticsModel
	List                *RuntimeListLookupModel
	Mutation            RuntimeMutationModel
	Hooks               RuntimeHookSetModel
	CreateFollowUp      RuntimeFollowUpModel
	UpdateFollowUp      RuntimeFollowUpModel
	DeleteFollowUp      RuntimeFollowUpModel
	AuxiliaryOperations []RuntimeAuxiliaryOperationModel
	OpenGaps            []RuntimeGapModel
}

// RuntimeLifecycleModel groups explicit state buckets used for condition mapping.
type RuntimeLifecycleModel struct {
	ProvisioningStates []string
	UpdatingStates     []string
	ActiveStates       []string
}

// RuntimeDeleteSemanticsModel captures explicit delete confirmation policy and states.
type RuntimeDeleteSemanticsModel struct {
	Policy         string
	PendingStates  []string
	TerminalStates []string
}

// RuntimeListLookupModel captures explicit list payload selection and item matching.
type RuntimeListLookupModel struct {
	ResponseItemsField string
	MatchFields        []string
}

// RuntimeMutationModel captures explicit mutable, force-new, and conflictsWith rules.
type RuntimeMutationModel struct {
	Mutable       []string
	ForceNew      []string
	ConflictsWith map[string][]string
}

// RuntimeHookSetModel groups imported helper hooks by CRUD phase.
type RuntimeHookSetModel struct {
	Create []RuntimeHookModel
	Update []RuntimeHookModel
	Delete []RuntimeHookModel
}

// RuntimeHookModel captures one imported helper hook.
type RuntimeHookModel struct {
	Helper     string
	EntityType string
	Action     string
}

// RuntimeFollowUpModel captures the explicit post-operation follow-up strategy.
type RuntimeFollowUpModel struct {
	Strategy string
	Hooks    []RuntimeHookModel
}

// RuntimeAuxiliaryOperationModel captures an imported provider operation beyond the primary CRUD binding.
type RuntimeAuxiliaryOperationModel struct {
	Phase            string
	MethodName       string
	RequestTypeName  string
	ResponseTypeName string
}

// RuntimeGapModel captures one open formal gap recorded alongside the generated runtime.
type RuntimeGapModel struct {
	Category      string
	StopCondition string
}

// TypeModel is one helper type emitted into a resource types file.
type TypeModel struct {
	Name     string
	Comments []string
	Fields   []FieldModel
}

// FieldModel is one renderable Go field in a generated spec type.
type FieldModel struct {
	Name     string
	Type     string
	Tag      string
	Comments []string
	Markers  []string
	Embedded bool
}

// PrintColumnModel is one kubebuilder printcolumn marker for a resource.
type PrintColumnModel struct {
	Name        string
	Type        string
	JSONPath    string
	Description string
	Priority    *int
}

// SampleModel renders one sample YAML for a generated resource.
type SampleModel struct {
	Body         string
	FileName     string
	MetadataName string
	Spec         string
}

// HasSpecField reports whether the generated spec exposes a Go field with the given name.
func (r ResourceModel) HasSpecField(name string) bool {
	for _, field := range r.SpecFields {
		if field.Name == name {
			return true
		}
	}
	return false
}
