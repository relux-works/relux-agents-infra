package infra

import (
	"errors"
	"fmt"
	"strings"
)

// ValidatePiExecutionEnvironment applies the Process-A denial contract on
// every platform. It never includes the rejected value in its error.
func ValidatePiExecutionEnvironment(environ []string) error {
	seen := map[string]bool{}
	for _, item := range environ {
		name, _, ok := strings.Cut(item, "=")
		if !ok || name == "" {
			return piError("pi_execution_environment_malformed", errors.New("malformed environment entry"))
		}
		if seen[name] {
			return piError("pi_execution_environment_malformed", errors.New("duplicate environment name"))
		}
		seen[name] = true
		if name == "HF_ENDPOINT" || name == "MODEL_ENDPOINT" || name == "GGML_BACKEND_PATH" || name == "LLAMA_API_KEY" {
			return piError("pi_execution_environment_invalid", fmt.Errorf("runtime-affecting environment name %q is denied", name))
		}
		upper := strings.ToUpper(name)
		for _, prefix := range []string{"DYLD_", "LD_", "NODE_", "BUN_", "LLAMA_ARG_"} {
			if strings.HasPrefix(upper, prefix) {
				return piError("pi_execution_environment_invalid", fmt.Errorf("runtime-affecting environment name %q is denied", name))
			}
		}
	}
	return nil
}
