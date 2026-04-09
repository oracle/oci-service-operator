package upgrade

type Report struct {
	FromVersion          string                `json:"fromVersion"`
	ToVersion            string                `json:"toVersion"`
	ComparedToOSOK       bool                  `json:"comparedToOSOK,omitempty"`
	Structs              []StructDiff          `json:"structs"`
	AllowlistSuggestions []AllowlistSuggestion `json:"allowlistSuggestions"`
}

type StructDiff struct {
	StructType    string        `json:"structType"`
	AddedFields   []FieldInfo   `json:"addedFields,omitempty"`
	RemovedFields []FieldInfo   `json:"removedFields,omitempty"`
	ChangedFields []FieldChange `json:"changedFields,omitempty"`
}

type FieldInfo struct {
	Name       string   `json:"name"`
	JSONName   string   `json:"jsonName,omitempty"`
	Mandatory  bool     `json:"mandatory"`
	Deprecated bool     `json:"deprecated"`
	ReadOnly   bool     `json:"readOnly"`
	UsedByOSOK bool     `json:"usedByOSOK,omitempty"`
	References []string `json:"references,omitempty"`
}

type FieldChange struct {
	FieldName  string    `json:"fieldName"`
	From       FieldInfo `json:"from"`
	To         FieldInfo `json:"to"`
	UsedByOSOK bool      `json:"usedByOSOK,omitempty"`
	References []string  `json:"references,omitempty"`
}

type AllowlistSuggestion struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}
