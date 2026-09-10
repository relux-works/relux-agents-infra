# BUG-260830-5pmaiz: composition-failure-degrades-silently-to-bare-launch

## Description
Observed on 2026-08-30: the installed project config lacked eight fields that later stories made required — max_segment_bytes, max_segments, restart_limit, restart_initial_backoff_seconds, restart_max_backoff_seconds, stable_run_seconds, quarantine_seconds and resource_pressure_mode. Every spawn then reported 'degraded_compose_failed ... bare child launch retained' as one line among its startup notes and proceeded. Several tracked runs executed without project MCP composition before an orchestrator noticed. Diagnosis also cost one invocation per missing field, because validation stops at the first absent field.

## Scope
agents-infra half only: the doctor/preflight diagnostic for a project profile (tools/agents-infra, project_config validation and the diagnostics command that reports it). The task-board half (prominent degraded flag in spawn result/status/observe) is skill-project-management BUG-260911-2tvq62.

## Acceptance Criteria
One diagnostic invocation (agents-infra doctor/preflight for the profile) reports every missing required field for the profile in a single run rather than only the first, with the field names and the config path; a negative test feeds a profile missing all eight late-added fields and asserts all eight are named; an existing valid profile reports none.
