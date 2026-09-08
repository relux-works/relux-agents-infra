# TASK-260707-1gnk6p review verdict — CR revision 3

Verdict: **changes requested**. Route to `to-dev`.

Reviewed candidate:

- Base: `436760d62f4ea451cf49614ff7e40109d96915b3`
- Candidate tree: `22dc42e2de0cef534373bcba0b300bb3f94def99`
- Changed paths: exactly `LOGBOOK.md`, `README.md`, `SKILL.md`, `scripts/setup.sh`, and `tools/agents-infra/setup_lldb_mcp_test.go`
- Foreign-path check: passed; no sibling delta survived the clean-base reimplementation.

## Blocking finding 1 — unusable external helper is accepted

Production call site: root `setup.sh` executes `scripts/setup.sh`; `install_lldb_mcp` accepts an external command at `scripts/setup.sh:131-153` based only on `command -v` and marker/path checks.

Adversarial macOS fixture placed an executable `lldb-mcp` on `PATH` that exits `42`, removed Homebrew from `PATH`, and ran the real `./setup.sh`. Observed:

- external helper exit status: `42`
- setup exit status: `0`
- setup printed `lldb-mcp already installed`
- setup printed `=== Done ===`

This is the `absent evidence treated as satisfied` / fail-open shape: command discovery is not usability evidence. It directly violates the acceptance criterion that unavailable prerequisites fail clearly rather than leave a broken opt-in behind successful setup. The same unvalidated external-helper acceptance exists in both the no-Homebrew branch (`scripts/setup.sh:134-138`) and the Homebrew-present/non-managed-path branch (`scripts/setup.sh:151-154`).

Evidence: `.temp/TASK-260707-1gnk6p-review/broken-external-01.log` and its disposable fixture output.

Required rework:

1. Define and enforce a bounded, behavior-based external-helper health contract that cannot report success for a helper which immediately fails. Do not turn a probe read/launch failure into absence or success.
2. Add production-entry negative tests for the unusable external helper through both acceptance branches (Homebrew absent and Homebrew present with the external helper outside the managed wrapper path).

## Blocking finding 2 — delegation bound is narrowed to one tested argument shape

`assertWrapperDelegates` at `tools/agents-infra/setup_lldb_mcp_test.go:147-157` invokes only `--socket auto`. In a disposable archive of the exact candidate, the generated wrapper was narrowed from `"$@"` forwarding to hard-coded `--socket auto`. Every `TestBootstrapSetup*` test still passed with `-count=1`.

This is the `narrowing, not deleting` shape: the suite proves one example, not verbatim argument-vector delegation.

Evidence: `.temp/TASK-260707-1gnk6p-review/mutant-arg-delegation-tests-01.log`.

Required rework: make the helper record an argument vector with multiple distinct shapes (including an empty argument and whitespace-bearing argument) and assert exact preservation through the generated production wrapper.

## Blocking finding 3 — helper failure propagation is untested

In another disposable candidate archive, the generated wrapper was narrowed from `exec "$helper" "$@"` to `"$helper" "$@" || true`, preserving the current fixture's stdout but swallowing helper failures. Every `TestBootstrapSetup*` test still passed with `-count=1`.

This is a bypass around delegation health: the suite checks successful output but never makes the helper fail and therefore cannot prove wrapper exit-status propagation.

Evidence: `.temp/TASK-260707-1gnk6p-review/mutant-swallow-helper-exit-tests-01.log`.

Required rework: configure the fake helper to emit a sentinel and return a non-zero status, execute the real generated wrapper, and assert both exact output and exact exit status.

## Blocking finding 4 — byte-identical no-rewrite claim is not proven

The idempotence test asserts only that the wrapper inode is stable (`tools/agents-infra/setup_lldb_mcp_test.go:188-204`). In a disposable candidate archive, the byte-identical branch was changed to `cp` the rendered bytes over the existing wrapper before removing the temporary file. This rewrote the wrapper while preserving its inode, and every `TestBootstrapSetup*` test still passed with `-count=1`.

Evidence: `.temp/TASK-260707-1gnk6p-review/mutant-same-inode-rewrite-tests-01.log`.

Required rework:

1. Prove the byte-identical branch performs no write, not merely no inode replacement (for example, preserve a fixed modification timestamp or make the already-executable identical fixture read-only and require the second production setup to succeed unchanged).
2. Record the forged wrapper inode before repair and assert divergent bytes are replaced by a different inode, alongside the existing behavior/delegation assertion.

## Passing evidence

- Exact delta has only the five allowed paths; `git diff --check` passed.
- `go -C tools/agents-infra test -count=1 -run '^TestBootstrapSetup' .` passed on the unmodified candidate.
- `zsh -n setup.sh scripts/setup.sh`, `gofmt -d tools/agents-infra/setup_lldb_mcp_test.go`, and `go -C tools/agents-infra vet .` passed.
- README and SKILL document `./setup.sh`, explicit `AGENTS_INFRA_SKIP_LLDB_MCP=1 ./setup.sh`, render/config inspection, restart, idempotence intent, and failure remediation.
- The forged early-`exit 0` wrapper is repaired by the current implementation, but the broader delegation and failure-propagation bounds above remain unproven.

The reviewer did not edit `LOGBOOK.md`: the reviewer role is repository-read-only. The producer should append these newly observed findings in the next revision.
