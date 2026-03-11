package apispec

import (
	"reflect"

	databasev1beta1 "github.com/oracle/oci-service-operator/api/database/v1beta1"
	mysqlv1beta1 "github.com/oracle/oci-service-operator/api/mysql/v1beta1"
	streamingv1beta1 "github.com/oracle/oci-service-operator/api/streaming/v1beta1"
)

type Target struct {
	Name       string
	SpecType   reflect.Type
	SDKStructs []string
}

var targets = []Target{
	{
		Name:     "AutonomousDatabases",
		SpecType: reflect.TypeOf(databasev1beta1.AutonomousDatabasesSpec{}),
		SDKStructs: []string{
			"database.CreateAutonomousDatabaseDetails",
			"database.UpdateAutonomousDatabaseDetails",
		},
	},
	{
		Name:     "MySqlDbSystem",
		SpecType: reflect.TypeOf(mysqlv1beta1.MySqlDbSystemSpec{}),
		SDKStructs: []string{
			"mysql.CreateDbSystemDetails",
			"mysql.UpdateDbSystemDetails",
		},
	},
	{
		Name:     "Stream",
		SpecType: reflect.TypeOf(streamingv1beta1.StreamSpec{}),
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
