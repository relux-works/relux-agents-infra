# TASK-260829-2t5xmi revision 2 rework

Address only reviewer finding F1 from `TASK-260829-2t5xmi_review-verdict.md` for accepted candidate revision 1.

## Required change

- Reject every configured seconds field that is converted with `time.Duration(seconds) * time.Second` when the value exceeds the maximum representable duration.
- Keep the bound explicit and overflow-safe in validation; do not add or change numeric defaults.
- Cover restart initial/max backoff, stable-run, quarantine, broker-start timeout, and any coupled supervision timeout whose conversion or ordering participates in the policy.
- Add a negative test through the real config/CLI resolution path using an otherwise-valid value just above the representable bound. It must refuse before runtime launch or ledger mutation.
- Preserve the already-reviewed restart/quarantine, half-open, stable-reset, manual-quarantine, and abrupt-client-death behavior.
- Record the regression and fix in `LOGBOOK.md`.

## Evidence

The reviewer proved that values around `10_000_000_000` seconds overflow `time.Duration`, making backoff negative and quarantine/stable timers immediate. The full evidence is attached to the task as `TASK-260829-2t5xmi_review-verdict.md`.
