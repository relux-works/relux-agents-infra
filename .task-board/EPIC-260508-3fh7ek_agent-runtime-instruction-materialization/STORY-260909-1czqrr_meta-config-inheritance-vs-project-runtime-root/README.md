# STORY-260909-1czqrr: meta-config-inheritance-vs-project-runtime-root

## Description
Separate two things agents-infra currently conflates: (1) where project configuration is inherited from, and (2) where the provider project surface is materialized. A parent directory that carries only .agents/.configs is a legitimate meta-config for every project beneath it; it must contribute configuration to those projects without becoming their runtime root.

## Scope
agents-infra project runtime root resolution, project-config inheritance, and the documentation of both.

## Acceptance Criteria
Config inheritance from a parent .agents/.configs works for nested projects; the provider surface is always materialized in the project directory itself; the inheritance contract and the meta-config pattern are documented.
