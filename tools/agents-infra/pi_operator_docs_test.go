package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPiOperatorContractDocumentsCycle10Boundary(t *testing.T) {
	root := sourceRepoRoot(t)
	readme := normalizedOperatorDoc(t, filepath.Join(root, "README.md"))
	for _, want := range []string{
		"### Managed Pi local-model operator contract",
		`profile = "qwen-3.8-27b"`,
		`yolo_mode = false`,
		`reasoning = true`,
		`thinking = "medium"`,
		`thinking_format = "qwen-chat-template"`,
		`pi_compatibility = "github-release:earendil-works/pi@v0.84.2:darwin-arm64#sha256-c996e888b7f7dce44bcf24f69176ac646c44139d3916bd49a6b28e5a8c5e3a65"`,
		`requested_capabilities = ["dflash", "text", "tools"]`,
		`target_argv = ["--model", "Muse-Glimmer-30B"]`,
		"lowercase_hex(SHA256(profile_bytes))",
		"profile_state_key_collision",
		"profile_state_path_invalid",
		"2f68ab1b3f28a9c4b8995f91984f8f47001a79735da7e57aa7fe6d223f90378b",
		"exactly 217 exhaustive regular-file records",
		"malicious same-UID process winning the bind race",
		"configured—not independently verified",
		"runtime-specific Muse benchmark or telemetry check",
		"backend catalogs, compiled observers, adapter/proxy layers, private-pipe authorities",
		"`LLAMA_ARG_*` environment names are refused before llama.cpp starts",
		"`HF_ENDPOINT` and `MODEL_ENDPOINT` are refused before llama.cpp starts",
		"Exact `GGML_BACKEND_PATH` is refused before managed state or runtime spawn",
		"llama.cpp build 10470 passes its inherited value to `dlopen()` during backend discovery",
		"Other `GGML_*` names remain outside this exact-name policy",
		"Exact `LLAMA_API_KEY` is refused before managed state or runtime spawn",
		"llama.cpp build 10470 uses it as the environment backing for `--api-key`",
		"`HF_TOKEN`, cache-location variables, and unrelated environment names remain admitted",
		"`LLAMA_API_KEY` values are never reported",
		"pi-infra --print-config",
		"`yolo_mode = true` injects",
		"Explicit `--no-approve`/`-na` conflicts with effective yolo",
		"logs/<UTC-start>-<random>.jsonl",
		"foreground-terminal ownership",
		"`restart_not_before` is the ledger's exact RFC3339 deadline or JSON `null`",
		"agents-infra runtime status --json.restart_not_before",
		"vendorplugin.LimitedUntil",
		"`last_failure` and `last_failure_at` are explicitly deferred",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README.md missing Pi operator contract fragment %q", want)
		}
	}
}

func TestReluxAgentsInfraSkillRoutesSafePiWorkflowToSource(t *testing.T) {
	skill := normalizedOperatorDoc(t, filepath.Join(sourceRepoRoot(t), "SKILL.md"))
	for _, want := range []string{
		"Pi launcher, setup, alias, catalog, and operator-contract changes belong in this source repository",
		"pi-infra --print-config",
		"Exact decoded UTF-8 profile bytes",
		"anchored no-follow containment contract",
		"requested/configured, never independently verified",
		"malicious same-UID process that wins the post-preflight bind race",
		"Do not reintroduce the retired cycle-7 backend/observer/proxy/attestation design",
		"`HF_ENDPOINT` and `MODEL_ENDPOINT`",
		"Exact `GGML_BACKEND_PATH`",
		"Other `GGML_*` names remain outside this exact-name policy",
		"Exact `LLAMA_API_KEY` is refused before managed state or runtime spawn",
		"Keep `HF_TOKEN`, cache-location variables, `LLAMA_API_KEY` lookalikes",
		"Non-`off` reasoning additionally requires profile `reasoning = true`",
		"Effective `yolo_mode = true` injects exactly one `--approve`",
		"`--approve`/`-a` controls one-run trust for project-local files",
		"mode-`0600` lifecycle JSONL",
		"foreground-terminal ownership",
		"`restart_not_before` is always present as the ledger's RFC3339 deadline or JSON `null`",
		"vendorplugin.LimitedUntil",
		"`last_failure` and `last_failure_at` remain explicitly absent",
	} {
		if !strings.Contains(skill, want) {
			t.Fatalf("SKILL.md missing managed Pi guidance %q", want)
		}
	}
}

func normalizedOperatorDoc(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return strings.Join(strings.Fields(string(data)), " ")
}
