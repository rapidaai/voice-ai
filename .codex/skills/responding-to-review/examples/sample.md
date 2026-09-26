# Review response example

- Comment: The unsupported-provider branch can return a nil implementation.
- Severity: Major.
- Disposition: Valid.
- Evidence: `path/factory.go:61` returns zero values; the regression test reproduces construction failure.
- Correction: Return the established error at the factory boundary.
- Verification: Focused package tests and scoped finalization pass.
- Reply: Fixed at the factory owner and covered by the unknown-provider regression test.
- Re-review: Required because the original finding was Major.
