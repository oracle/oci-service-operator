package apispec

import (
	"reflect"

	v1beta1 "github.com/oracle/oci-service-operator/api/v1beta1"
)

type Target struct {
	Name       string
	SpecType   reflect.Type
	SDKStructs []string
}

var targets = []Target{
	{
		Name:     "AutonomousDatabases",
		SpecType: reflect.TypeOf(v1beta1.AutonomousDatabasesSpec{}),
		SDKStructs: []string{
			"database.CreateAutonomousDatabaseDetails",
			"database.UpdateAutonomousDatabaseDetails",
		},
	},
	{
		Name:     "MySqlDbSystem",
		SpecType: reflect.TypeOf(v1beta1.MySqlDbSystemSpec{}),
		SDKStructs: []string{
			"mysql.CreateDbSystemDetails",
			"mysql.UpdateDbSystemDetails",
		},
	},
	{
		Name:     "Stream",
		SpecType: reflect.TypeOf(v1beta1.StreamSpec{}),
		SDKStructs: []string{
			"streaming.CreateStreamDetails",
			"streaming.UpdateStreamDetails",
		},
	},
}

func Targets() []Target {
	result := make([]Target, len(targets))
	copy(result, targets)
	return result
}
