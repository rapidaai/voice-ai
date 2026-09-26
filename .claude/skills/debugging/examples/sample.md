# Debugging example

- Expected: Unknown provider selection returns the established unsupported-provider error.
- Actual: Construction continues with a nil implementation.
- Reproduction: `go test ./path/to/package -run TestUnknownProvider`
- Hypothesis: The factory's default branch returns no error.
- Evidence: The default branch at `path/factory.go:61` returns zero values.
- Regression proof: The new test fails before the fix with a nil implementation.
- Root cause: The selection owner omitted the fallback error.
- Verification: Focused package tests and scoped finalization pass.
