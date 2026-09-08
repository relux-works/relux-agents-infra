# TASK-260909-z10c1u: document-config-inheritance-and-meta-configs

## Description
Document how agents-infra resolves configuration and where it materializes provider surfaces, and state the meta-config pattern as supported rather than accidental.

Cover: the ancestor walk used for project-config resolution and how a nested project inherits from an ancestor .agents/.configs/project-config.toml; that the provider surface is always materialized in the project directory; precedence between global ~/.agents, an ancestor meta-config, and a project-local .agents; which files a meta-config directory may legitimately contain (.configs only, with no install receipt) and why that is not a broken install; how to inspect the effective resolution with agents-infra doctor local, whose *_source fields name the file each value came from.

Worked example: a directory such as ~/src holding .agents/.configs/project-config.toml so that every repository beneath it shares one primary-session policy, while each repository keeps its own surface.

## Scope
README.md and SKILL.md in the agents-infra source repo; keep both in sync with the behavior BUG-260909-163sup establishes.

## Acceptance Criteria
README.md and SKILL.md describe config inheritance, surface materialization, precedence, and the meta-config pattern with a worked example; the documented behavior matches the shipped code and is stated as a supported pattern.
