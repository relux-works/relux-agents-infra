# STORY-260824-260ddq: integrate-vendor-targets-on-current-main

## Description
Carry the independently accepted vendor/environment/model target candidate onto the current main branch after the original managed Story workspace became stale.

## Scope
Mechanical current-main reconciliation and final integration of accepted candidate tree 95d12fb4; preserve current trunk changes and all accepted vendor-target behavior.

## Acceptance Criteria
The fresh Story candidate contains the complete accepted vendor-target implementation plus all current-main changes, differs only where intended, passes the accepted validation gates, and integrates through task-board without integration_base_moved or content loss.
