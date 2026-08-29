# Board completeness and justified-gap record — revision 6 addendum

Supplements `TASK-260825-imfmgz_decomposition-verification.md`. Nothing in that
document is withdrawn.

## Board delta for revision 6

**No board element was created, removed, split or re-scoped.** The story keeps
its three leaves and its two dependency links:

```
STORY-260825-1r7z9o
  ├── TASK-260825-imfmgz  design-shared-runtime-owner-contract   (blocks lsojra)
  ├── TASK-260825-lsojra  implement-shared-runtime-leases        (blocks 1lc8o7)
  └── TASK-260825-1lc8o7  prove-two-pi-one-qwen-runtime
```

Revision 6 closes one blocking finding inside an existing leaf's deliverable. It
adds no requirement the specification did not already carry, so decomposition
rule 3 does not apply: there is **no justified-gap element**, because there is no
new element.

## Checklist delta

Four items added to `TASK-260825-lsojra`, all carrying revision-6 obligations to
the implementer. Each supersedes or extends a named earlier item rather than
duplicating one, following the convention items 13, 20, 21, 22, 25, 27, 29, 30
and 33 already established on that task:

| Item | Supersedes | Traces to |
| --- | --- | --- |
| 36 — closure at the decoder, B12 step 4, renumbered steps 5 and 6 | item 30's frame handling | spec §6.2 B11/B12 step 4, §11 |
| 37 — test 12.2.30c and the six decoder mutants, both duplicate orders, non-discriminating rows recorded | extends item 34 | spec §12.2.30c, §12.3 |
| 38 — protocol version 6 | item 33's version number | spec §9 |
| 39 — wiring proof over the decoded key multiset | item 35 | spec §12.4 |

No item was added to `TASK-260825-1lc8o7`: revision 6 changes nothing about the
acceptance proof of §13 — two RUN handles, two Pi pids, one attested runtime pid,
overlap, and bounded cleanup on all six surfaces are untouched.

## Spec traceability for this leaf

`TASK-260825-imfmgz` traces to the story's own AC clause "*Broker lifetime, lease
recovery, final-release cleanup, status, explicit stop, and negative
identity/concurrency tests are documented*". Revision 6 sits under the last of
those: the authorization frame is the launcher's identity gate, and a gate whose
field set cannot be established is a negative identity test that does not exist.

## Self-verification against the out-of-scope list

Checked `§15` (scope boundaries) and `§14` (rejected alternatives) before making
the change, per decomposition rule 3:

- `§15` excludes migrating other project configurations, remote or multi-user
  brokers, and non-`mlx_lm.server` runtimes. The decoder boundary touches none of
  them: it is a rule about bytes already crossing an existing pipe between two
  processes the specification already defines.
- `§14` rejects putting the exec plan in the authorization frame. Revision 6 does
  not re-open that — it *narrows* the frame further, and the rejection's reasoning
  (a frame carrying a plan makes the pipe a command channel) is strengthened by a
  decoder that refuses any member outside the closed five.
- `§9` already required a protocol bump for a gate-set change. The bump to `6` is
  the existing rule applied, not new scope.

Result: no new scope, no gap-closing element, and no research task — the review's
finding was a defect in an existing deliverable with a determinate fix, not an
open question.

## Proportionality

Three leaves for a story that specifies a protocol, implements it, and proves it
on real processes. Revision 6 does not change that judgement, and deliberately
adds no fourth leaf for the decoder work: it is four checklist items on the leaf
that already owns the launcher.
