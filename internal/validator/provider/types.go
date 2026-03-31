package provider

type UsageKind string

const (
	UsageKindCompositeLiteral      UsageKind = "composite_literal"
	UsageKindGeneratedRequestField UsageKind = "generated_request_field"
	UsageKindPostLiteralAssignment UsageKind = "post_literal_assignment"
)

type FieldUsage struct {
	StructType string
	FieldName  string
	File       string
	Line       int
	Kind       UsageKind
}

type Analysis struct {
	SourcePath string
	Usages     []FieldUsage
}
