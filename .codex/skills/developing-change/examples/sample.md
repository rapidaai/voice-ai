# Development handoff example

- Tier: Standard
- Acceptance criteria: Supported selection is unchanged; unsupported selection returns the established error.
- Allowed paths: `path/factory.go`, `path/factory_test.go`.
- Test-first evidence: `TestUnknownProvider` failed with a nil implementation before the fix.
- Implementation: Added the missing fallback return in the factory owner.
- Verification: Focused package tests and `just agent-finalize` passed.
- Review boundary: Working tree diff against `HEAD`.
- Remaining risks: None known.
