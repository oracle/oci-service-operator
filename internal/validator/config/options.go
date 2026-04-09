package config

import (
	"fmt"
	"strings"
)

type Options struct {
	ProviderPath  string
	AllowlistPath string
	Service       string
	Verbose       bool
	Format        string
	BaselinePath  string
	WriteBaseline string
	FailOnNew     bool
	UpgradeFrom   string
	UpgradeTo     string
}

func DefaultOptions() Options {
	return Options{
		ProviderPath:  ".",
		AllowlistPath: "validator_allowlist.yaml",
		Format:        "table",
	}
}

func (o Options) Validate() error {
	if strings.TrimSpace(o.ProviderPath) == "" {
		return fmt.Errorf("provider path must not be empty")
	}
	return nil
}

func (o Options) ValidateUpgrade() error {
	if strings.TrimSpace(o.UpgradeFrom) == "" || strings.TrimSpace(o.UpgradeTo) == "" {
		return fmt.Errorf("upgrade mode requires both upgrade-from and upgrade-to")
	}
	return nil
}

func (o Options) HasAllowlist() bool {
	return strings.TrimSpace(o.AllowlistPath) != ""
}

func (o Options) HasBaseline() bool {
	return strings.TrimSpace(o.BaselinePath) != ""
}

func (o Options) WantsBaselineWrite() bool {
	return strings.TrimSpace(o.WriteBaseline) != ""
}

func (o Options) WantsUpgrade() bool {
	return strings.TrimSpace(o.UpgradeFrom) != "" || strings.TrimSpace(o.UpgradeTo) != ""
}

func (o Options) HasServiceFilter() bool {
	return strings.TrimSpace(o.Service) != ""
}
