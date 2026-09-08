# TASK-260831-1pfnxx: invert-the-pi-plugin-to-describe-agents-infra-from-within

## Description
The first-consumer probe found a structural inversion, not a missing method. The shipped skill-agents-management v0.5.0 pi plugin describes agents-infra from the OUTSIDE: pkg/agentic/systems/pi/binary.go resolves the agents-infra binary on the launch environment's PATH, proven by the probe error 'launchenv: agents-infra not found in the launch environment's PATH', and pkg/agentic/systems/pi/args.go emits 'pi --profile <profile> --'. That models agents-infra as an external tool the caller launches, which is correct for task-board and wrong for agents-infra itself, which IS agents-infra. The probe also showed pi registered as an agentic system but absent as a runtime: lookup runtime 'pi' returns found=false, as does 'local-models'. Closing this needs typed named plan variants with a managed-host versus client partition, a producer-side composition contract, and a pi x local-models runtime — an agents-management release that the accepted lockstep plan currently forbids without a new independent acceptance, since it keys every compatibility row to exact v0.5.0.

## Scope
(define task scope)

## Acceptance Criteria
The pi plugin expresses both perspectives without either consumer special-casing it: a managed host that IS agents-infra and a client that launches it. pi resolves as a runtime, and the pi x local-models runtime exists. A producer-side composition contract lets agents-infra contribute composition rather than only consume identity. agents-infra can then remove its hardcoded pi branch, currently 40 pi_ files in internal/infra plus explicit branches in canonical_target.go lines 84, 275 and 421, primary_session_launch_plan.go line 254, and project_config.go line 406. The lockstep release plan is re-accepted for the extra release before any of it lands, because the current plan requires both sides to compile exact v0.5.0.
