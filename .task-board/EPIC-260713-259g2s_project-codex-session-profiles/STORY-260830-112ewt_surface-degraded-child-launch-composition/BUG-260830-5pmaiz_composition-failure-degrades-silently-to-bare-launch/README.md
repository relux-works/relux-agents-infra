# BUG-260830-5pmaiz: composition-failure-degrades-silently-to-bare-launch

## Description
Observed on 2026-08-30: the installed project config lacked eight fields that later stories made required — max_segment_bytes, max_segments, restart_limit, restart_initial_backoff_seconds, restart_max_backoff_seconds, stable_run_seconds, quarantine_seconds and resource_pressure_mode. Every spawn then reported 'degraded_compose_failed ... bare child launch retained' as one line among its startup notes and proceeded. Several tracked runs executed without project MCP composition before an orchestrator noticed. Diagnosis also cost one invocation per missing field, because validation stops at the first absent field.

## Scope
(define bug scope / affected area)

## Acceptance Criteria
A degraded composition is surfaced prominently enough that an orchestrator does not spawn further children without deciding, and one diagnostic invocation reports every missing required field for the profile rather than only the first.
