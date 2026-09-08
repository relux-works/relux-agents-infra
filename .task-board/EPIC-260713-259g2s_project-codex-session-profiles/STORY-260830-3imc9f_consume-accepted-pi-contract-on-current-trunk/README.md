# STORY-260830-3imc9f: consume-accepted-pi-contract-on-current-trunk

## Description
Implement the concrete agents-infra consumer of the independently accepted agents-management observer and Pi Process-A contract on a fresh protected-trunk workspace, isolated from the stale lockstep-planning Story.

## Scope
agents-infra generic plugin consumption, sanitized MLX observer, Pi Process-A/result bridge, fake Process-B composition, dependency pin, docs and static/fake tests only. No live runtime, model, process, service, socket, endpoint or user configuration access.

## Acceptance Criteria
1. Fresh selected base equals fetched protected origin/main. 2. Exact reviewed agents-management commit 046baef is consumed without local replace or copied interfaces. 3. Concrete observer and Pi turn/result bridge satisfy TASK-260830-y6infr AC with Process-B ownership retained. 4. Static/fake full, race, vet, build, cross-platform, mutation and no-live-runtime gates pass. 5. Independent review accepts and canonical PR merges before activation.
