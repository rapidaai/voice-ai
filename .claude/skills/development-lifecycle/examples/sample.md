# Lifecycle routing example

- Tier: Standard
- Outcome: Correct the provider selection fallback without changing the public contract.
- Acceptance criteria: Known providers select their implementation; unknown providers return the established fallback error.
- Allowed paths: Factory package and its package-level test.
- Risks: Incorrect fallback behavior and missing factory coverage.
- Skills: `change-analysis`, matching integration skill, `developing-change`, `reviewing-change`.
- Verification: Focused package test, skill validator, scoped `just agent-finalize`.
- Delivery: Not requested.
