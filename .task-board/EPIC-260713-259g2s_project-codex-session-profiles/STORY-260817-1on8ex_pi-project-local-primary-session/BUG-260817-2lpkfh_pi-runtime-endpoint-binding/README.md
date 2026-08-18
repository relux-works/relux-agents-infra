# BUG-260817-2lpkfh: pi-runtime-endpoint-binding

## Description
Bind the managed runtime listen host and port to the declared loopback base_url so compose cannot report or probe a different backend than the direct child it launches.

## Scope
Shared Pi primary-session config parser, launch-plan resolver, production-entry negative tests, setup/install verification, and contract documentation. Preserve the trusted-runtime threat model while rejecting endpoint divergence in supported managed profiles.

## Acceptance Criteria
Production compose refuses wildcard bind and runtime-port drift; accepted argv expresses the exact 127.0.0.1 base_url endpoint; tests exercise installed/production entry paths; documentation states the invariant; focused and full Go validation plus setup/install/verify pass.
