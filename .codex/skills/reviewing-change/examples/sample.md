# Review example

- Boundary: Merge base `abc123`, head `def456`, clean worktree.
- Acceptance criteria reviewed: Supported and unsupported provider selection.
- Verification considered: Focused package test and scoped finalization.
- Finding: Major, `path/factory.go:61`, unsupported selection returns a nil implementation, causing construction to fail later without provider context.
- Disposition: Valid.
- Remedy: Return the established unsupported-provider error at the factory boundary and add a regression test.
- Decision: Changes requested until the Major finding is resolved and reverified.
