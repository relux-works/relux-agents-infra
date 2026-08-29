# TASK-260824-1glviz — reviewer verdict (CR-TASK-260824-1glviz-1 rev 1)

- Verdict: **accepted**
- Reviewer run: `RUN-260824-9cfd3b` (claude-opus-5 / high), goal-bound: no
- Reviewed delta: base `2f74fb0c757c3d3038d744a054e0ce1b68656df7` -> candidate tree `dd84306b4db3edf703c787971f24bca6bc09e81e`, 20 paths, `repository_delta=present`
- Reviewer modified no reviewed file. Post-review worktree tree recomputed as
  `dd84306b4db3edf703c787971f24bca6bc09e81e` (identical to the reviewed candidate).
  All mutation work ran in throwaway copies under `/tmp`, since removed.

## 1. Reconciliation identity (AC: "semantically equals the merge")

Independent reconstruction, not a re-read of the producer's claim:

```
ACC=$(git commit-tree 95d12fb4… -p cf21665…)      # 2b9ddd477aacc92c34cea16a54b0fcb01302c22d
git merge-tree --write-tree --messages main $ACC  # rc=1 -> f2926828fb804ffa31eeefaf7992257343362d3b
git diff f2926828… dd84306b…                      # LOGBOOK.md only, 6 deletions
```

- `git merge-base(main, ACC)` = `cf21665…`, the declared authoritative base.
- Git's own three-way merge auto-merged `README.md`, `SKILL.md`, `infra.go`,
  `infra_test.go` cleanly and conflicted **only** on `LOGBOOK.md`.
- The candidate tree is **byte-identical to that merge on every path except
  `LOGBOOK.md`**, where the sole difference is removal of the 6 conflict-marker
  lines (`<<<<<<< main` / `=======` / `>>>>>>>` x2). No content resolution
  choice was made beyond marker removal.
- Blob-level classification over the union of all three trees:
  16 blobs == accepted candidate only, 4 blobs == neither side (the four
  three-way merges), 19 blobs == main only (16 of them `.task-board/**`, plus
  `.configs/codex-config.toml`, `internal/infra/codex_config.go`,
  `setup_test.go`). Zero board paths in the CR. This reproduces the producer's
  "16 / 3 / 4" claim exactly.

## 2. Current-main preservation (AC: "fast-profile work is preserved")

Line-level check on all five overlap files (`README.md`, `SKILL.md`, `infra.go`,
`infra_test.go`, `LOGBOOK.md`): every non-blank line **added** by main relative
to `cf21665…` is present in the candidate, and no line main **deleted** is
resurrected.

| File | main-added | missing from candidate | main-deleted | resurrected |
| --- | ---: | ---: | ---: | ---: |
| README.md | 8 | 0 | 2 | 0 |
| SKILL.md | 7 | 0 | 0 | 0 |
| internal/infra/infra.go | 4 | 0 | 1 | 0 |
| internal/infra/infra_test.go | 132 | 0 | 8 | 0 |
| LOGBOOK.md | 12 | 0 | 0 | 0 |

`LOGBOOK.md` in the candidate is byte-identical to the accepted candidate's
`LOGBOOK.md` **and** a pure-insertion superset of main's (66 additions, 0
deletions) — the retention the task authorises.

## 3. Validation rerun by this reviewer

| Command | Result |
| --- | --- |
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `go test ./... -count=1` | **EXIT=0**; 112.122s / 2.885s / 187.797s (log `.temp/review-1glviz/go-test-full-01.log`) |
| `agents-infra verify global` | rc=0 |
| `agents-infra verify local /Users/alexis/src/casual-talks` | rc=0 |
| `lsof -nP -iTCP:18011 -sTCP:LISTEN` | **rc=1, empty** — no listener |
| `pgrep -af 'mlx_lm\.server'` | **rc=1, empty** — no runtime process |

No `agents-infra` test binary remained after the suite.

## 4. Gates attacked, not read

### 4a. Live production entry point (installed aliases, `casual-talks`)

Installed `~/.local/bin/{openai,anthropic,qwen}-infra` are regular 0755 files
carrying the candidate wrapper body (`-f`, `!-L`, `-x` sibling guard, then
`exec "$TARGET" target <entrypoint> "$@"`); installed `agents-infra` advertises
`target ENTRYPOINT` and `compose --entrypoint`, so the reviewed code is what is
installed. Thirteen probes through the real aliases:

- refused `target_identity_conflict`: `-mgpt-4o`, `-m=gpt-4o`, `--model=gpt-4o`,
  `-pwork`, `--config=profile="work"`, `--model=claude-sonnet-5`,
  `--effort=low`, `--thinking high`, `--provider openai`,
  `--endpoint http://evil:1/v1`, and a post-wrapper-delimiter `--model`
- refused `unknown_entrypoint`: `agents-infra target bogus-infra` (no legacy fallback)
- admitted (rc=0): exact repeat `--model gpt-5.6-sol`, and the operand form once
  a second delimiter reaches the provider

Delimiter semantics are fail-closed at the alias boundary: Go's flag parser eats
one `--`, the wrapper eats one, so a bare `-- -- --model X` is still locked and
only `-- -- -- --model X` reaches Codex as an operand.

### 4b. Mutation testing (narrowing, not delete-only) on a throwaway copy

| Mutant | Killed? | By |
| --- | --- | --- |
| M1 entrypoint-vendor gate narrowed to `openai-infra` only | **SURVIVED** | — |
| M1b same gate deleted entirely | killed | `TestCanonicalEntrypointResolutionNeverInfersOrFallsBack/alias_vendor_mismatch` |
| M2 Pi reasoning assertion narrowed (admits `"high"`) | killed | `TestCanonicalQwenProfileAssertionsFailClosed/reasoning_mismatch` |
| M3 Codex attached `-m<value>` narrowed to `-model` prefix | **SURVIVED** | — |
| M4 `-L` symlink guard dropped from the POSIX alias wrapper | **SURVIVED** | — |
| M5 `requireExactTargetValue` narrowed to `EqualFold` | **SURVIVED** | — |
| M6 Setup canonical-config gate inverted (`ModeLocal`->`ModeGlobal`) | killed | `TestCanonicalConfigurationFailurePreventsSetupAndVerifyMutation` |

For every surviving mutant I proved the **unmutated production path still
refuses**, so none is a defect:

- M3/M5: a probe harness over `BuildCanonicalTargetLaunchPlan` showed all 12
  attached/equals divergent Codex forms (`-mX`, `-m=X`, `--model=X`,
  `--model-reasoning-effort=X`, `-pX`, `-p=X`, `--profile=X`, `-c=…`,
  `--config …`, `--config=profile=…`) return `target_identity_conflict`;
  reconfirmed live in 4a.
- M4: a symlinked sibling makes the installed wrapper exit 127 with
  `missing or non-regular sibling`.
- M1: the gate is a single uniform `map[entrypoint]` comparison inside one loop
  with no per-entrypoint branching, so one covered entrypoint does bind the
  production behaviour; only the committed table is single-instance.

Production call sites are named in the accepted tests
(`runTarget -> BuildCanonicalTargetLaunchPlan -> lockCanonicalTargetArguments`,
`runCompose -> … -> buildPiPrimarySessionLaunchPlan`), and
`TestRunTargetDispatchPreservesCallerCWDAndLocksBeforeProviderSideEffects`
proves the fake `codex` on `PATH` is never reached on identity conflict or on an
unconfigured alias — the lock precedes provider side effects.

## 5. Why accepted despite four surviving mutants

The four survivors are **test-table breadth gaps inherited from the already
accepted `CR-TASK-260824-2a4gk3-2`**, not defects introduced by this
reconciliation, and this task's charter is explicit: "Do not redesign accepted
behavior." Each corresponding guard was independently proven to hold at the real
production entry point, so no positive-path-only claim is being accepted. The
task's own AC — semantic merge equality, all accepted tests plus three alias
provenance checks green, current-main fast-profile work preserved, no
listener/process/lock residue — is fully met and independently reproduced above.

## 6. Follow-up (non-blocking, for a later hardening leaf)

1. `TestCanonicalCodexSelectorsAcceptExactAndRefuseEveryDivergentForm` claims
   "every divergent form" but enumerates only the six space-separated/`-c` forms.
   Add the attached and `=` short forms (`-mX`, `-m=X`, `--model=X`, `-pX`,
   `-p=X`, `--profile=X`, `--model-reasoning-effort=X`, `-c=`, `--config`,
   `--config=`) so M3 dies.
2. `TestCanonicalAliasRefusesMissingAndNonRegularSibling` covers three states;
   add a `symlinked` state so the `-L` guard (M4) is pinned.
3. `alias vendor mismatch` covers only `openai-infra`; table it over all three
   entrypoints so M1 dies.
4. Consider a case-divergence case (`GPT-5.6-SOL`) to pin exact-equality (M5).
