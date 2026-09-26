# Change analysis example

- Today: The factory selects the provider in `path/factory.go:42`; unsupported values return the error declared at `path/factory.go:61`.
- Owner: The factory package owns selection; the provider owns transport behavior.
- Consumers: Assistant construction and the factory tests.
- Contracts: No public schema change.
- Proposed scope: Factory file plus its package test.
- Exclusions: Provider internals and UI configuration.
- Risk: An unsupported value could select a zero implementation.
- Evidence needed: Add a regression test for the unsupported value.
- Recommended tier: Standard.
