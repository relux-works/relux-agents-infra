# BUG-260817-161m6u — Reviewer Verdict Cycle 2: ACCEPTED

Reviewer run `RUN-260817-92fe50` (not goal-bound). Both cycle-1 blockers are
closed with evidence I reproduced independently; nothing new surfaced.

## Cycle-1 Finding 1 (installed launcher lacked the gate) — CLOSED

`~/.local/bin/agents-infra` is now Mach-O mtime 2026-08-17 23:12,
SHA-256 `3cd24eab3fc674d31310fa1cf59a1e723ef61f718c4b466dec742f747eb36a0d`
(was `df62cd52…f42c3` at cycle-1 review). `strings` counts: `LLAMA_ARG_` 1,
`runtime-affecting` 1, `DYLD_` 1, `pi_execution_environment_invalid` 1.

String parity is the weak half. The behavioral proof I ran myself, driving the
**installed** `~/.local/bin/pi-infra` shim (`exec agents-infra pi`) against a
disposable managed-profile project, isolated `HOME`, `env -i`, and the official
Pi asset first on `PATH`
(`.temp/BUG-260817-161m6u/cycle2/probe/run2.sh`):

| Probe | Exit | stderr | Managed state under isolated `HOME` |
| --- | ---: | --- | --- |
| control (clean env) | 1 | `/bin/echo` printed its runtime argv, then `runtime exited before readiness` | `Caches/agents-infra/pi/<state-key>/<profile-key>` CREATED |
| `LLAMA_ARG_MODEL=SECRETMODELPATH42` | 1 | `runtime-affecting environment name "LLAMA_ARG_MODEL" is denied` | none — only `Caches/` |
| `LLAMA_ARG_CTX_SIZE=SECRETCTX99` | 1 | `runtime-affecting environment name "LLAMA_ARG_CTX_SIZE" is denied` | none — only `Caches/` |
| `llama_arg_model=SECRETLOWER77` (lowercase) | 1 | `runtime-affecting environment name "llama_arg_model" is denied` | none |

`grep -c SECRET` over both stdout and stderr of every denied probe is 0, so no
inherited value leaks. The control is what gives the negatives teeth: with a
clean environment the same installed binary walks the whole chain — argv plan,
env gate, inbound `PI_*` gate, `VerifyPiExecutionIdentity` against the real
asset, runtime identity, state tree creation — and actually spawns the runtime
child. The denied runs stop strictly before the state tree exists. This is the
ordering claim proved on the artifact the operator executes, not on a fresh
`go build`.

Caveat recorded honestly: an equivalent `DYLD_INSERT_LIBRARIES` probe through
the shim cannot be run this way — macOS SIP strips `DYLD_*` when exec'ing the
`/usr/bin/env sh` shim, so that variable never reaches the process. The `DYLD_`
gate stays covered at source level (mutant below), not through the shim.

`~/.agents` runtime tree is byte-identical to the repo for the three changed
files (`pi_catalog.go`, `pi_operator_docs_test.go`, `README.md`), so the
local-install lane — whose wrapper rebuilds from `AGENTS_INFRA_SOURCE_DIR` —
also carries the gate.

The corrected artifact wording now states what each row establishes
(`setup global` refreshes `~/.agents`, not the bootstrap-owned executable;
`verify global` does not establish executable freshness). The proxy-signal
finding from cycle 1 is resolved.

## Cycle-1 Finding 2 (documented contract ungated) — CLOSED

`pi_operator_docs_test.go:28` now pins
``"`LLAMA_ARG_*` environment names are refused before llama.cpp starts"``.
Mutants run in a disposable repo copy (`.temp/BUG-260817-161m6u/cycle2/mutant/`):

| README mutant | `TestPiOperatorContractDocumentsCycle10Boundary` |
| --- | --- |
| Baseline | ok |
| Delete the whole deny sentence | FAIL |
| Narrow: drop only `` `LLAMA_ARG_*` `` from the documented list | FAIL |
| Weaken ordering: `before llama.cpp starts` → `after` | FAIL |
| Restore | ok |

Delete-only would have proved little; the narrowing and ordering mutants show
the assertion binds the exact class and the exact ordering claim.

## Source gate re-attacked on the cycle-2 tree

Same disposable copy, with the gitignored Pi asset symlinked in so the
production-entry negatives RUN rather than SKIP (verified: all six subtests
`duplicate`, `loader`, `llama model`, `llama absent option`, `inbound agent dir`,
`inbound sessions` executed):

| Mutant on `pi_catalog.go:296` | Result |
| --- | --- |
| Delete `"LLAMA_ARG_"` from the prefix list | FAIL |
| Narrow to `"LLAMA_ARG_MODEL"` | FAIL (`LLAMA_ARG_CTX_SIZE` admitted) |
| Restore | ok |

Independent full verification in the working checkout:

| Command | Exit |
| --- | ---: |
| `go test ./... -count=1` (main 68.4s, attachments 2.9s, infra 107.4s) | 0 |
| `go vet ./...` | 0 |
| `gofmt -l .` (empty) | 0 |

## AC / DoD status

| Item | State |
| --- | --- |
| Refuses `LLAMA_ARG_*` before spawn, no values leaked | met (validator, production `RunPi`, and installed binary) |
| Focused tests: MODEL, second name, exact clean control, existing gates | met (`pi_test.go:460-528`) |
| Shared Pi launch documentation describes the contract | met and now gated |
| Source tests, setup/install refresh, installed local verification | met — bootstrap `./setup.sh` rebuilt the installed binary, behavioral proof attached |
| Gate attacked, not read | met — code, docs, and installed-runtime mutants above |

## Non-blocking, carried forward (not part of this AC)

- Production Pi negatives still `t.Skipf` when the gitignored asset
  `.temp/TASK-260817-2h8hn4/pi-standalone-darwin-arm64-0.84.2` is absent, and the
  package still reports `ok`. Story-wide fixture convention (LOGBOOK 2026-08-17
  2132), not introduced here. They executed in this review.
- llama.cpp also honours non-`LLAMA_ARG_` names (`LLAMA_CACHE`, `HF_TOKEN`,
  `HF_ENDPOINT`). Outside this AC; worth a separate bug.
- Denying lowercase `llama_arg_*` is broader than POSIX requires; harmless.

## Handoff

Reviewer-archetype run supplies no `commit_ack`. Acceptance evidence is this
artifact plus `BUG-260817-161m6u_review-evidence-cycle2.tgz`; the commit-owning
mover commits the scope and makes the final `done` transition with
`commit_ack=scope_committed`.
