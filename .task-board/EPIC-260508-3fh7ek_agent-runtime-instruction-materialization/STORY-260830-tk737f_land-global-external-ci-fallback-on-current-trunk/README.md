# STORY-260830-tk737f: land-global-external-ci-fallback-on-current-trunk

## Description
Replay the reviewed external-CI fallback policy onto exact current trunk, harden the exclusive unrepairable-external trigger against broadened policy mutants, and deliver through fresh review and PR.

## Scope
Versioned workflow source, source-to-installed parity test, README, and LOGBOOK only. Preserve current-trunk instruction changes and keep fast-mode config activation separate until board schema lands.

## Acceptance Criteria
Fresh exact origin/main selected; revision 4 semantic delta preserved; focused Setup-path test asserts the complete only-when externally caused and agent-unrepairable trigger; broadened repairable-or-inconvenient mutant fails; full uncached Go suite and vet pass under serialized validation; independent review accepts; canonical setup/install parity passes after merge.
