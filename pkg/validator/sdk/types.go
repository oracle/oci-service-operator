package sdk

import "reflect"

type Target struct {
	QualifiedName string
	PackageName   string
	TypeName      string
	ImportPath    string
	ReflectType   reflect.Type
}

type SDKStruct struct {
	QualifiedName string
	PackageName   string
	TypeName      string
	ImportPath    string
	Fields        []SDKField
}

type SDKFieldKind string

const (
	SDKFieldKindScalar    SDKFieldKind = "scalar"
	SDKFieldKindStruct    SDKFieldKind = "struct"
	SDKFieldKindInterface SDKFieldKind = "interface"
)

type SDKField struct {
	Name                     string
	Type                     string
	JSONName                 string
	Mandatory                bool
	Deprecated               bool
	ReadOnly                 bool
	Documentation            string
	Kind                     SDKFieldKind
	NestedFields             []SDKField
	InterfaceImplementations []SDKImplementation
}

type SDKImplementation struct {
	QualifiedName string
	PackageName   string
	TypeName      string
	ImportPath    string
	Fields        []SDKField
}
