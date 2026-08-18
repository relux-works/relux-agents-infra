# Reviewer Verdict — Cycle 4

Task: `TASK-260817-2h8hn4`
Verdict: changes requested
Route: `analysis`

## Finding

### F1 — Verified Pi package bytes are not bound to the executed Node program

The cycle-four catalog closes the prior version-only identity gap, but direct managed launch still resolves and executes the catalogued npm `dist/cli.js` entrypoint. That published entrypoint begins with `#!/usr/bin/env node`. The contract authenticates the Pi package tree while leaving the Node interpreter, loader-affecting environment, installed dependency closure, and verification-to-exec binding unspecified.

This is a `bypass path around the check` and a `check present but uncalled from production` at the composed-artifact boundary: package verification can remain `verified` while the program that receives the normalized argv is changed before Pi's catalogued parser observes it.

Concrete reproduction on the review host:

1. A shebang-equivalent Node entrypoint received the safe managed argv `--provider local-qwen --model local-qwen/qwen-3.8-27b` unchanged in the control run.
2. With the package/entrypoint bytes still unchanged, inherited `NODE_OPTIONS=--require=<preload>` loaded caller-controlled JavaScript before the entrypoint. The preload rewrote `process.argv` to `--provider attacker --model attacker/model`.
3. Output is saved in `.temp/TASK-260817-2h8hn4/reviewer-node-options-bypass-01.log`. The probe and preload are task-scoped scratch evidence beside it.

The mechanism is production-faithful for the selected catalog entry:

- the extracted official npm `0.84.2` `dist/cli.js` starts with `#!/usr/bin/env node`;
- Node documents that `NODE_OPTIONS` is interpreted before command-line options and permits `--require`; the required module runs before the program entrypoint;
- section 6 reports only environment values set by the wrapper and no section rejects or removes `NODE_OPTIONS`, binds an exact Node executable instead of `/usr/bin/env`, or authenticates the resolved dependency closure;
- section 7 verifies Pi before starting the project-selected runtime, then starts Pi only after runtime readiness/attestation. The contract does not bind the later exec to the previously observed bytes, so a package replacement during this interval is another untested verification/use path.

The acceptance suite does not name inherited loader injection, hostile `node` resolution through `PATH`, dependency-tree substitution, or mutation between verification and Pi spawn. Its fake catalogued Pi therefore proves the package verifier and argv bridge separately, not the composed production artifact.

## Required rework

Define one deterministic managed execution closure and its production enforcement before implementation:

1. Bind the catalog to what is actually executed, not only the npm package directory. A clean option is an exact reviewed standalone Pi executable. If Node remains the host, specify and verify the exact interpreter plus every code-bearing dependency that can run before or alter the parser, and invoke the interpreter by its verified absolute path rather than through the package shebang and `PATH`.
2. Specify the managed Pi environment allow/deny contract. Loader/config injection such as `NODE_OPTIONS`, `NODE_PATH`, alternate loader/import hooks, and equivalent host-runtime variables must be rejected or removed before Pi starts; diagnostics must report names/provenance without values. Native passthrough may retain its existing environment semantics.
3. Close the verification/use interval. The contract must establish that the bytes/environment authorized by the gate are the bytes/environment used by the Pi child after runtime readiness, or explicitly narrow the claim and threat boundary without describing an unbound package observation as executable identity.
4. Add production-entry negative cases that keep the Pi package tree byte-exact while injecting a preload, place a hostile `node` first on `PATH`, substitute one resolved dependency, and mutate/swap the entrypoint after initial verification but before Pi spawn. Each must refuse before Pi reaches managed state. Add a narrowing mutant that verifies only the package tree while omitting host/environment binding and require a named test to fail.
5. Update the TOML/catalog diagnostics, failure codes, rejected alternatives, process order, and downstream implementation task acceptance criteria to carry this composed-execution identity rule.

## Checks completed

- Re-read the complete decision artifact and confirmed byte parity with its task outcome resource (`3019cef3aad1b656f258d086c102eb19caf6301bf8cbf8fbef7f964d1027bb13`).
- Rechecked official Pi models, settings/trust, usage/model-option, custom-model, and session precedence documentation on current `main`.
- Rechecked the immutable npm `0.84.2` tarball identity, `dist/cli.js` shebang, package manifest evidence, parser hash, and prior reviewer findings.
- Reproduced caller-controlled argv replacement through Node's documented preload mechanism without modifying any verified package byte.
- Confirmed exact provider/model matching, fake-separator handling, DFlash nonce/PID attestation, unknown-versus-absence rules, process ownership, three-task dependency chain, and task/resource traceability otherwise remain explicit and proportional.
- No product build/test applies to this no-implementation decision task. The failure is contract research rework, not a human-only or external blocker.

Official references:

- https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/models.md
- https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/settings.md
- https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/usage.md
- https://nodejs.org/api/cli.html#node_optionsoptions
