# TASK-260831-oobgci: read-mapped-memory-in-process

## Description
Fix the memory instrument at its design rather than its calibration. The scored quantity's mapped component takes 11 distinct values spanning 2.26-3.64 MB on the baseline, about 0.01 percent of a 29 GB footprint, and 3 distinct values on the candidate, effectively static once the GGUF is mapped. A 2.2-5.8 s external vmmap fork at 0.2 Hz, policed by a 7.0 s bound, exists to track a rounding error on one runtime and a constant on the other, and it is that fork which refused 268/288 and 179/200 of the mapped observations in the 2026-08-31 pair.

## Scope
(define task scope)

## Acceptance Criteria
The mapped component is read in-process via proc_pidinfo or a mach_vm_region walk, so both memory components come from one read at one cadence. The 7.0 second mapped bound is removed rather than retuned, and the Mach bound stops being a separate cause of refusal for context_75k. Merely re-deriving the cost term upward does not satisfy this task and is refused as confirmation bias. A production-entry test shows a scored memory value for a realistic scenario window on both runtimes, and is red against the current external-fork implementation.
