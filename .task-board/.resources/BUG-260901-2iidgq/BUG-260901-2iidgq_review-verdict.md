# BUG-260901-2iidgq review verdict: ACCEPTED

## Independent verification performed

- Read full CR diff (project_config.go, claude_launch.go, codex_launch.go, tests, docs).
- Confirmed provider-local parsing: loadPrimaryProviderProjectConfig routes codex/claude launch through parseProjectConfigForProvider, which parses only the selected hosted provider's [agents.<provider>.primary_session] plus shared mcp/agents-root TOML, leaving Pi profiles/targets/entrypoints unparsed for that call.
- Confirmed all Pi-owning call sites (canonical_target.go, pi_plan.go, pi_launch_posix.go, pi_platform_windows.go, pi_lifecycle_legacy.go, pi_standalone.go, pi_shared_resolve.go, project_config_setup.go, infra.go, child_launch_composition.go) still use the original strict loadCompositeProjectConfig — untouched by this change.
- go build ./... and go test ./... (full suite, all packages) pass.
- Adversarial mutation: reverted the provider-local Pi/targets/entrypoints guard to always-parse (the exact pre-fix crossover bug). TestBuildPrimarySessionLaunchPlanProviderLocalIgnoresIncompletePiProfile immediately failed on both codex and claude subtests with the publisher error, proving the negative test is load-bearing and not positive-path-only. Reverted the mutation afterward; file matches CR exactly (git diff --stat clean).
- Independently reproduced the real bsim-shaped scenario end-to-end (not just trusting the producer's note): copied the real /Users/alexis/src/.agents/.configs/project-config.toml (with the actual qwen-3.8-27b-mlx-8bit profile), stripped its publisher line, and ran the candidate CLI via go run:
  - compose --agent codex: exit 0, status ok
  - compose --agent claude: exit 0, status ok
  - compose --agent pi: exit 1, invalid_project_configuration, names agents.pi.profiles.qwen-3.8-27b-mlx-8bit.publisher
  This matches AC1-AC3 exactly against production config shape, not just synthetic fixtures.
- Confirmed AC4 negative coverage: malformed selected-provider fields (codex model=false, claude yolo_mode as string) and malformed shared fields (mcp.enabled_servers wrong type, agents root wrong type) all still fail with the correct field name, per TestBuildPrimarySessionLaunchPlanSelectedAndSharedPolicyRemainStrict.
- Confirmed canonical qwen-infra path still validates strictly via TestCanonicalQwenSelectedProfileStillRequiresPublisher (BuildCanonicalTargetLaunchPlan uses the strict composite loader).
- README.md/SKILL.md/LOGBOOK.md updates are accurate and consistent with the actual code behavior.

## Verdict

Accepted. Fix is minimal, provider-local validation is correctly scoped, negative tests are proven load-bearing via mutation, and the bsim reproduction was independently verified against real config data, not just trusted from the producer's notes.