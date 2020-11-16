package validate

import "testing"

func TestEnvVarName(t *testing.T) {
	if !EnvVarName("VALID_NAME") ||
		!EnvVarName("VALID_123") ||
		EnvVarName("INVALID!") ||
		EnvVarName("Invalid_Name") ||
		EnvVarName("INVALID-NAME") {
		t.Error("Environment variables not correctly validating.")
	}
}
