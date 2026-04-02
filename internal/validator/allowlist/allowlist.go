package allowlist

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Status string

const (
	StatusUsed                 Status = "used"
	StatusIntentionallyOmitted Status = "intentionally_omitted"
	StatusPotentialGap         Status = "potential_gap"
	StatusFutureConsideration  Status = "future_consideration"
)

type Allowlist struct {
	Structs map[string]Struct `yaml:"structs" json:"structs,omitempty"`
}

type Struct struct {
	Fields map[string]Field `yaml:"fields" json:"fields,omitempty"`
}

type Field struct {
	Status     Status   `yaml:"status" json:"status,omitempty"`
	Reason     string   `yaml:"reason,omitempty" json:"reason,omitempty"`
	References []string `yaml:"references,omitempty" json:"references,omitempty"`
}

func Load(path string) (Allowlist, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return Allowlist{}, err
	}
	var result Allowlist
	if err := yaml.Unmarshal(contents, &result); err != nil {
		return Allowlist{}, err
	}
	if result.Structs == nil {
		result.Structs = map[string]Struct{}
	}
	for name, strct := range result.Structs {
		if strct.Fields == nil {
			strct.Fields = map[string]Field{}
		}
		result.Structs[name] = strct
	}
	return result, nil
}
