# TASK-260721-2c1847: remove-python-attachments-helper

## Description
Port the agents-attachments helper into the Go agents-infra CLI and remove the runtime Python dependency. Scope includes materialize, list, path, and stage-images behavior parity; a backwards-compatible agents-attachments entrypoint backed by the Go binary; setup/install linkage; documentation; and Go tests replacing the Python helper tests.

## Scope
(define task scope)

## Acceptance Criteria
The agents-infra binary exposes attachment helper commands equivalent to the existing agents-attachments materialize, list, path, and stage-images flows. The installed agents-attachments command no longer requires python3 and delegates to the Go implementation. Setup/doctor/link tests validate the new installation shape. Existing attachment workflow documentation no longer lists Python as a required project tool. Go tests cover manifest lookup, path resolution, image staging/copying, HEIC converter fallback behavior, redacted staged names, and error cases formerly covered by the Python tests.
