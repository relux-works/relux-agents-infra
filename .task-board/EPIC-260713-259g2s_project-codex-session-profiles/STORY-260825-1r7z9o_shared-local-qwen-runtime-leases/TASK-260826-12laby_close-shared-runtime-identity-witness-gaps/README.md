# TASK-260826-12laby: close-shared-runtime-identity-witness-gaps

## Description
Add symmetric adversarial witnesses for executable identity and protocol-version refusal classes discovered by final Story review.

## Scope
Tests and evidence only unless a witness reveals a production defect. Add same-device/wrong-inode coverage for all four Dev/Ino production comparisons, add below-current protocol-version coverage, and make the known positive-control timeout event-driven if this can be done without widening product behavior. Preserve the accepted shared-runtime production contract.

## Acceptance Criteria
Each Dev/Ino identity gate is independently killed when its inode comparison is removed; protocol versions both above and below the exact current version are refused and exact-version control passes; named witnesses run at the real production entry points; focused, race, full package, build, vet, and formatting checks pass or any known flake is reproduced and honestly classified; review accepts the CR before Story integration.
