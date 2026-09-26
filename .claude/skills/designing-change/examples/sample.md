# Design decision example

- Context: Unsupported providers currently reach construction without a useful error.
- Constraint: Preserve the public request shape and provider implementations.
- Chosen option: Return the established error from the factory default branch.
- Rejected option: Add validation in every caller, which duplicates selection policy.
- Owner: The factory remains the single selection owner.
- Failure behavior: Unsupported values fail immediately with provider context.
- Verification: Existing selections plus unknown-provider regression coverage.
- Rollback: Revert the isolated factory change.
