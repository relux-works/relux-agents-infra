# TASK-260830-1jpse1: generalize-plugin-graph-in-agents-management

## Description
Replace the fixed two-layer agentic-then-vendor hierarchy in skill-agents-management with a general plugin graph in which a plugin declares its kind and its dependencies on other plugins, so a new kind — inference engine, agent environment, model vendor, weight artifact or a kind not yet imagined — is added without adding a layer.

## Scope
skill-agents-management as the owning repository. Contract gaps are fixed and released there rather than worked around in consumers.

## Acceptance Criteria
Plugin kind and inter-plugin dependencies are declared data rather than positions in a fixed hierarchy, the registry resolves and validates the dependency graph including refusing cycles and unsatisfiable declarations, the six shipped agentic-system plugins and four vendor plugins keep working unchanged, and adding a kind requires no change to the registry contract.
