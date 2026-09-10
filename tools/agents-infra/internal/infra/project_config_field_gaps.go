package infra

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pelletier/go-toml/v2"
)

// ProjectProfileFieldGap names one required field that is absent from a
// project's Pi profile. loadCompositeProjectConfig and its parsers stop at
// the first absent field by design (BUG-260830-5pmaiz keeps that fail-closed,
// single-error behavior for the production launch path). This scan is a
// separate, additive, presence-only pass so one doctor/preflight invocation
// can name every gap across a profile instead of forcing one fix-and-rerun
// cycle per missing field.
type ProjectProfileFieldGap struct {
	ConfigPath string
	Profile    string
	Field      string
	Expected   string
}

// projectPiProfileFieldGaps re-reads every ancestor project config with a
// lenient TOML unmarshal (not the strict parser) and reports every absent
// required field for every declared Pi profile, tagged with its dotted field
// path and source config path. A config that fails to parse as TOML at all is
// skipped here; that failure is already surfaced by the strict parse error
// returned alongside this scan's result.
func projectPiProfileFieldGaps(ancestors []string, globalProjectConfigPath string) ([]ProjectProfileFieldGap, error) {
	var gaps []ProjectProfileFieldGap
	for _, dir := range ancestors {
		path := filepath.Join(dir, ".agents", ".configs", projectConfigFileName)
		if globalProjectConfigPath != "" && samePath(path, globalProjectConfigPath) {
			continue
		}
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read project config %s: %w", path, err)
		}
		var document map[string]any
		if err := toml.Unmarshal(data, &document); err != nil {
			continue
		}
		agentsTable, ok := document["agents"].(map[string]any)
		if !ok {
			continue
		}
		piTable, ok := agentsTable["pi"].(map[string]any)
		if !ok {
			continue
		}
		profilesTable, ok := piTable["profiles"].(map[string]any)
		if !ok {
			continue
		}
		for name, raw := range profilesTable {
			profileTable, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			gaps = append(gaps, missingPiProfileFields(profileTable, path, name)...)
		}
	}
	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].ConfigPath != gaps[j].ConfigPath {
			return gaps[i].ConfigPath < gaps[j].ConfigPath
		}
		if gaps[i].Profile != gaps[j].Profile {
			return gaps[i].Profile < gaps[j].Profile
		}
		return gaps[i].Field < gaps[j].Field
	})
	return gaps, nil
}

// missingPiProfileFields mirrors the required-field schema enforced by
// parsePiProfile, parsePiLifecycleLogRetention, parsePiRuntime, and
// parsePiRuntimeSharing, but only checks presence (never type or cross-field
// validity) and collects every absent field instead of returning on the
// first one.
func missingPiProfileFields(table map[string]any, path, name string) []ProjectProfileFieldGap {
	field := "agents.pi.profiles." + name
	var gaps []ProjectProfileFieldGap
	add := func(dottedField, expected string) {
		gaps = append(gaps, ProjectProfileFieldGap{ConfigPath: path, Profile: name, Field: dottedField, Expected: expected})
	}
	requirePresent := func(t map[string]any, dottedField, key, expected string) {
		if _, present := t[key]; !present {
			add(dottedField+"."+key, expected)
		}
	}

	for _, scalar := range []struct{ key, expected string }{
		{"provider", "non-empty string"},
		{"publisher", "non-empty string"},
		{"family", "non-empty string"},
		{"model", "non-empty string"},
		{"base_url", "non-empty string"},
		{"api", `string equal to "openai-completions"`},
		{"reasoning", "boolean"},
		{"input", `array equal to ["text"]`},
		{"context_window", "positive integer"},
		{"max_tokens", "positive integer not exceeding context_window"},
		{"thinking", "documented Pi thinking level string"},
		{"requested_capabilities", `array of strings, e.g. ["text", "tools"]`},
	} {
		requirePresent(table, field, scalar.key, scalar.expected)
	}

	retentionField := field + ".lifecycle_log_retention"
	if raw, present := table["lifecycle_log_retention"]; !present {
		add(retentionField, "table")
	} else if retentionTable, ok := raw.(map[string]any); ok {
		for _, key := range []string{
			"max_count", "max_bytes", "max_envelope_bytes", "max_age_seconds",
			"create_timeout_seconds", "append_timeout_seconds", "close_timeout_seconds",
			"status_timeout_seconds", "maintenance_timeout_seconds", "max_scan_entries",
			"max_scan_control_bytes", "max_mutations_per_operation",
		} {
			requirePresent(retentionTable, retentionField, key, "positive integer")
		}
	}

	if _, present := table["compat"]; !present {
		add(field+".compat", "table")
	}

	runtimeField := field + ".runtime"
	raw, present := table["runtime"]
	if !present {
		add(runtimeField, "table")
		return gaps
	}
	runtimeTable, ok := raw.(map[string]any)
	if !ok {
		return gaps
	}
	for _, scalar := range []struct{ key, expected string }{
		{"executable", "absolute NUL-free path string"},
		{"argv", "non-empty array of strings"},
		{"readiness_path", `string equal to "/models"`},
		{"startup_timeout_seconds", "positive integer"},
		{"shutdown_timeout_seconds", "positive integer"},
	} {
		requirePresent(runtimeTable, runtimeField, scalar.key, scalar.expected)
	}

	if raw, present := runtimeTable["sharing"]; present {
		sharingField := runtimeField + ".sharing"
		if sharingTable, ok := raw.(map[string]any); ok {
			for _, scalar := range []struct{ key, expected string }{
				{"mode", `string equal to "exclusive" or "shared"`},
				{"linger_seconds", "non-negative integer seconds"},
				{"max_leases", "positive integer"},
				{"max_segment_bytes", "positive integer"},
				{"max_segments", "positive integer"},
				{"heartbeat_interval_seconds", "positive integer seconds, less than lease_stale_seconds"},
				{"lease_stale_seconds", "positive integer seconds"},
				{"broker_start_timeout_seconds", "positive integer seconds"},
				{"restart_limit", "positive integer"},
				{"restart_initial_backoff_seconds", "positive integer seconds"},
				{"restart_max_backoff_seconds", "positive integer seconds, at least restart_initial_backoff_seconds"},
				{"stable_run_seconds", "positive integer seconds"},
				{"quarantine_seconds", "positive integer seconds"},
				{"resource_pressure_mode", `string equal to "disabled" or "provider"`},
			} {
				requirePresent(sharingTable, sharingField, scalar.key, scalar.expected)
			}
			if mode, _ := sharingTable["resource_pressure_mode"].(string); mode == "provider" {
				pressureField := sharingField + ".resource_pressure"
				if pressureRaw, present := sharingTable["resource_pressure"]; !present {
					add(pressureField, "table")
				} else if pressureTable, ok := pressureRaw.(map[string]any); ok {
					for _, scalar := range []struct{ key, expected string }{
						{"observation_path", "non-empty string"},
						{"observation_timeout_milliseconds", "positive integer"},
						{"pressure_threshold_bytes", "positive integer"},
						{"recovery_threshold_bytes", "positive integer"},
						{"eviction_grace_seconds", "non-negative integer seconds"},
						{"pressure_action", "non-empty string"},
						{"unknown_action", "non-empty string"},
						{"busy_action", "non-empty string"},
					} {
						requirePresent(pressureTable, pressureField, scalar.key, scalar.expected)
					}
				}
			}
		}
	}

	if raw, present := runtimeTable["dflash"]; present {
		dflashField := runtimeField + ".dflash"
		if dflashTable, ok := raw.(map[string]any); ok {
			for _, scalar := range []struct{ key, expected string }{
				{"target_model", "non-empty string equal to the profile model"},
				{"draft_model", "non-empty string"},
				{"target_argv", "non-empty array of strings"},
				{"draft_argv", "non-empty array of strings"},
			} {
				requirePresent(dflashTable, dflashField, scalar.key, scalar.expected)
			}
		}
	}

	return gaps
}
