# TASK-260817-3a0zr3 reviewer verdict — cycle 1

Verdict: **changes requested** → `to-dev`

## Finding F1 — managed alias type drift bypasses production verification

Severity: acceptance-blocking integrity bypass.

Negative-evidence shape: **bypass path around the check**.

`piInfraLauncherFailures` reads the launcher and then calls `os.Stat` at
`tools/agents-infra/internal/infra/runtime_receipt.go:181-195`. `os.Stat`
follows symlinks, so the advertised regular-file check examines the symlink
target rather than the managed alias path itself.

Production-entry reproduction:

1. Run source-built `agents-infra setup local /tmp/<project> --source-dir <repo>`.
2. Copy the installed `.local/bin/pi-infra` bytes outside the project.
3. Replace `.local/bin/pi-infra` with a symlink to that external copy.
4. Run source-built `agents-infra verify local /tmp/<project>`.

Observed: `verify local` exits `0` and prints `verified local agent runtime`.
The alias is no longer the regular managed artifact setup installed, yet the
verification gate reports it valid. This defeats AC 1/6 and the documented
claim that missing, incorrect, or drifted alias artifacts are detected.

Required rework:

- Inspect the alias path itself without following links and refuse every
  symlink/non-regular replacement in both setup reconciliation and verify.
- Add a production setup/verify regression that replaces the installed alias
  with a symlink to byte-identical content. A content-only narrowing must make
  the named test fail.
- Audit the sibling target path for the same follow-link ambiguity and either
  enforce the intended exact artifact type or document/test an explicitly
  permitted link policy.

## Finding F2 — setup does not repair documented mode drift

`installPiInfraLauncher` returns early when bytes match at
`tools/agents-infra/internal/infra/infra.go:977-983`, without checking the
artifact type or mode. On a production-installed alias changed from `0755` to
`0644`, `setup local` logs `Pi launcher already up to date`, then its own
postcondition fails because mode is `0644`.

This contradicts `README.md:135-137` and `SKILL.md:258-260`, which say setup
repairs alias drift while setup/verify reject changed mode.

Required rework:

- Make setup repair every managed alias drift class it claims to repair,
  including mode and type, or narrow the documentation consistently with the
  actual supported contract.
- Add production setup-entry coverage for mode repair and the symlink case;
  helper-only body tests are insufficient.

## Validation

- `go test ./... -count=1`: pass.
- `git diff --check`: pass.
- `task-board validate`: pass.
- These positive gates do not cover the reproduced symlink bypass.

No product/setup code was modified by the reviewer. The finding was recorded
in `LOGBOOK.md` under `2026-08-17 1848`.
