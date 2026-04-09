package formal

import "testing"

func requirePlantUML(t *testing.T) {
	t.Helper()
	if _, err := plantUMLBinary(); err != nil {
		t.Skip(err.Error())
	}
}
