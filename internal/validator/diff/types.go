package diff

type FieldStatus string

const (
	FieldStatusUsed                 FieldStatus = "used"
	FieldStatusIntentionallyOmitted FieldStatus = "intentionally_omitted"
	FieldStatusPotentialGap         FieldStatus = "potential_gap"
	FieldStatusFutureConsideration  FieldStatus = "future_consideration"
	FieldStatusDeprecated           FieldStatus = "deprecated"
	FieldStatusReadOnly             FieldStatus = "read_only"
	FieldStatusUnclassified         FieldStatus = "unclassified"
)

type Report struct {
	Structs []StructReport `json:"structs"`
}

type StructReport struct {
	StructType string        `json:"structType"`
	Coverage   Coverage      `json:"coverage"`
	Fields     []FieldReport `json:"fields"`
}

type Coverage struct {
	EligibleFields int     `json:"eligibleFields"`
	UsedFields     int     `json:"usedFields"`
	Percent        float64 `json:"percent"`
}

type FieldReport struct {
	StructType     string      `json:"structType"`
	FieldName      string      `json:"fieldName"`
	FieldType      string      `json:"fieldType"`
	JSONName       string      `json:"jsonName"`
	Mandatory      bool        `json:"mandatory"`
	Used           bool        `json:"used"`
	Deprecated     bool        `json:"deprecated"`
	ReadOnly       bool        `json:"readOnly"`
	Status         FieldStatus `json:"status"`
	Reason         string      `json:"reason,omitempty"`
	References     []string    `json:"references,omitempty"`
	Documentation  string      `json:"documentation,omitempty"`
	PreviousStatus FieldStatus `json:"previousStatus,omitempty"`
}
