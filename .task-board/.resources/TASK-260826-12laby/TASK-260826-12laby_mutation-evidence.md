# TASK-260826-12laby mutation evidence

All mutants were applied one at a time to a scratch extraction of `HEAD` plus
the candidate test patch. Every gate used `-count=1`. Each production file was
restored from a byte copy and checked with `cmp` before the next mutant.

| Production call site | Narrowing mutant | Named witness | Exit | Discriminating failure |
| --- | --- | --- | ---: | --- |
| `sharedBrokerServer.attestClient` (`pi_shared_broker_darwin.go:836`) | remove `clientExec.Ino` comparison | `TestSharedBrokerAttestClientRejectsEveryGateDeleteAndNarrowWitness/client_executable_same_device_wrong_inode` | 1 | Gate returned no refusal. |
| `connectAndAttestSharedRuntime` (`pi_shared_client_darwin.go:341`) | remove `peerIdentity.Ino` comparison | `TestConnectAndAttestSharedRuntimeRejectsEveryGateDeleteAndNarrowWitness/broker_executable_same_device_wrong_inode` | 1 | Gate returned no refusal. |
| `sharedRuntimeBrokerCandidates` (`pi_shared_operator_darwin.go:363`) | remove `identity.Ino` comparison | `TestSharedRuntimeBrokerCandidatesRejectSameDeviceWrongInodeAtProductionEntry` | 1 | Copied executable was returned as a broker candidate. |
| `stopRecordedSharedRuntimeWithDependencies` (`pi_shared_operator_darwin.go:470`) | remove `brokerIdentity.Ino` comparison | `TestSharedRuntimeForceStopRejectsRecordedBrokerIdentityNarrowingBeforeSignal/broker_executable_same_device_wrong_inode` | 1 | Wrong-inode evidence reached the signal path and ended in `runtime_shutdown_timeout`. |
| `sharedBrokerServer.attestClient` (`pi_shared_broker_darwin.go:839`) | refuse only `ProtocolVersion > current` | exact control + `past_protocol_version_range_narrowing` | 1 | Exact control passed; past-version witness failed because the gate admitted it. |
| `connectAndAttestSharedRuntime` (`pi_shared_client_darwin.go:379`) | refuse only `ProtocolVersion > current` | exact control + `past_protocol_version_range_narrowing` | 1 | Exact control passed; past-version witness failed because the gate admitted it. |
| `RunSharedRuntimeLauncher` authorization comparison (`pi_shared_launcher_darwin.go:72`) | accept `ProtocolVersion <= current` | exact control + `past_protocol_version` | 1 | Exact control passed; past-version frame executed the target and produced no refusal JSON. |

Clean focused groups exited `0`. The focused changed-scope race suite exited
`0` in `41.469s`.
