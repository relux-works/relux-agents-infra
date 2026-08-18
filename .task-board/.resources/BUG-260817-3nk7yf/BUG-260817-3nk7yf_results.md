# BUG-260817-3nk7yf — Cycle 5 implementation evidence

## Outcome

The setup-owned skill-link validator now proves directory-graph acyclicity, not
only per-link containment. It follows contained directory-link targets with an
explicit DFS `visiting`/`done` state: re-entry is refused as a transitive cycle,
while multiple links to an already completed target remain a valid DAG.

Production call sites remain:

- setup preflight before destination mutation: `infra.Setup` -> `validateSourceSkillLinks`
- setup postcondition and `verify local`: `runtimeArtifactFailures` -> `managedSkillLinkFailures`

The global ownership boundary is preserved: top-level provider packages are
selected exactly as before; every descendant reachable from a setup-managed
package is inspected.

## Regression and narrowing evidence

- Initial production-entry transitive-cycle tests: exit 1, both setup and verify
  admitted `.skills/transitive-cycle-probe -> ../cycle-target` plus
  `cycle-target/back -> ../.skills/transitive-cycle-probe`.
  Log: `transitive-red-02.log`.
- Focused production setup/verify suite after the fix: exit 0.
  Log: `focused-tests-06.log`.
- Per-link-only narrowing mutant (directory target traversal removed): exit 1;
  both named production-entry transitive-cycle tests failed by admitting the
  cycle. Log: `transitive-narrowing-mutant-01.log`.
- Byte-restored graph traversal plus transitive and DAG control: exit 0.
  Log: `transitive-restored-01.log`.
- The first post-fix focused run exited 1 because canonical DFS paths used
  `/private/var` while the layout retained `/var`; the lexical containment
  check was corrected to accept either spelling before canonical identity is
  proven. The rerun above exited 0. Log: `focused-tests-05.log`.

## Validation gates

Every command below ran directly; exit codes are the actual process statuses.

| Gate | Exit | Evidence |
| --- | ---: | --- |
| Focused installed-binary production suite | 0 | `focused-tests-06.log` |
| `go test ./... -count=1` | 0 | `full-tests-05.log` |
| `go vet ./...` | 0 | `go-vet-05.log` |
| `go build ./...` | 0 | `go-build-05.log` |
| Focused `gofmt -d` plus empty-output assertion | 0 / 0 | `gofmt-diff-05.log` |
| Source-built CLI build | 0 | `source-binary-build-02.log` |
| Pristine `/tmp` setup / verify / `find -L` | 0 / 0 / 0 | `pristine-setup-07.log`, `pristine-verify-05.log`, `pristine-find-follow-05.log` |
| Global source setup / verify | 0 / 0 | `global-setup-06.log`, `global-verify-06.log` |
| Source project setup / verify / `find -L` | 0 / 0 / 0 | `source-project-setup-06.log`, `source-project-verify-06.log`, `source-project-find-follow-06.log` |
| `local-models` source setup / verify / `find -L` | 0 / 0 / 0 | `local-models-setup-06.log`, `local-models-verify-06.log`, `local-models-find-follow-06.log` |
| `local-models` literal-dir, canonical skill target, and `pi-infra` assertions | 0 | direct compound assertion |
| Codex / Claude `--print-config` | 0 / 0 | `local-models-codex-config-06.log`, `local-models-claude-config-06.log` |
| Source / `local-models` `git diff --check` | 0 / 0 | `git-diff-check-05.log`, `local-models-diff-check-05.log` |
| Source / `local-models` cached-index emptiness | 0 / 0 | direct `git diff --cached --quiet` |
| `task-board validate` | 0 | `board-validate-05.log` |

One intentionally invalid pristine destination was first placed beneath the
source checkout; setup correctly refused self-sync with exit 1. The valid
pristine gate was rerun outside the source at
`/tmp/BUG-260817-3nk7yf-pristine-zuJAVq` and passed.
Before that rerun, one invocation used the binary's initially miscomputed temp
path and exited 127; the binary was rebuilt into the task-scoped repo `.temp`
path (`source-binary-build-02.log`) before any runtime acceptance evidence was
collected.

`local-models` retains Codex `gpt-5.6-sol/high/yolo=true` and Claude
`claude-opus-5/yolo=true`. No literal `$AGENTS_INFRA_SOURCE_DIR` directory is
present, `skills/relux-agents-infra` resolves to the contained
`.skills/relux-agents-infra` package, and recursive following inspection exits
zero.

Nothing was staged or committed.
