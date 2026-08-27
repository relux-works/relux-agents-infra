/// The exact upstream revisions this prototype was built and measured against.
///
/// Recorded in the binary, echoed on the `listening` event, and used as the
/// `system_fingerprint` on every completion, so a captured smoke transcript
/// always names the revision that produced it.
///
/// Keep in sync with `Package.swift` and `Package.resolved`.
let mlxSwiftRevision = "0.31.6 (0bb916c67f4b9e5c682cbe02a42c701c93ab5021)"
let mlxSwiftLMRevision = "3.31.4 (bd4b7434e6bdb588c7ef55706ff8904cb7fd4c57)"
