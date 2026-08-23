# RFC 0005: Refine Authentication Middleware Contracts

- Status: Accepted
- Date: 2026-08-23
- Owner: Platform API

## Summary

Refine the credential-specific middleware introduced by RFC 0004 without changing which
credential classes are enabled or how authentication is selected. Add an explicit HTTP error
response type, export the shared authentication failure message, use the repository validator
package for empty, nil, and zero/range validation, and rename command tests so their filenames
describe authentication middleware configuration rather than generic wiring.

## Scope

Production changes are limited to `pkg/middlewares`. Command packages change only by renaming
their existing authentication wiring test files; command production wiring is unchanged.

## Design

### Authentication Error Contract

Add exactly this response type:

```go
type AuthenticationError struct {
	Error string `json:"error"`
}
```

Export the immutable shared transport message with the exact declaration
`const AuthenticationFailureMessage = "Invalid authentication credentials"`. Gin middleware returns
`AuthenticationError{Error: AuthenticationFailureMessage}` and gRPC middleware uses the same
exported message. Safe internal log messages remain package-private constants.

### Validation

Use `pkg/validator` for middleware validation with these exact substitutions:

- Preserve current source selection. User Gin values continue to fall back only when the
  higher-priority raw value is exactly empty; project, organization, and service Gin values
  continue to fall back when the higher-priority value is blank. Use `validator.NonZero` for the
  former and `validator.NotBlank` for the latter.
- Trim the selected value at the existing boundary, then use `validator.NotBlank` for final
  credential presence and completeness checks.
- Use `validator.NonNil` for resolvers, loggers, returned principles, nested claim information,
  existing context identity, and optional delegated project IDs. A typed-nil interface is
  intentionally treated as nil and fails safely instead of risking a panic.
- Use `validator.Between(userID, uint64(1), uint64(math.MaxInt64))` for parsed user IDs.
- Use `validator.NonZero(selectedProjectID)` for selected project IDs, preserving the existing
  acceptance of values above `math.MaxInt64`.

String trimming remains at the credential boundary before values are passed to authenticators.
Parsing errors and authentication errors remain explicit error checks. No new validation helper
is added to middleware. Whitespace-only user path/header values retain their current precedence,
while whitespace-only scoped-key header/query values retain their current fallback behavior.

### Test Names

Rename each `cmd/*/wiring_test.go` file changed by RFC 0004 to
`cmd/*/authentication_middleware_test.go`. Test behavior and package ownership remain unchanged.

## Compatibility

Authentication behavior, middleware constructor signatures, status codes, JSON field names,
gRPC messages, credential precedence, and service wiring remain unchanged. Exporting the shared
message is additive.

## Verification

- Update middleware tests to assert the typed HTTP error response and exported message.
- Add regression coverage for whitespace source precedence, numeric identifier boundaries, and
  typed-nil resolver/principle/context values.
- Run middleware and command authentication tests.
- Run scoped `go vet`, broad backend tests, and `git diff --check`.
- Confirm no old `authenticationFailureMessage` reference or command `wiring_test.go` remains.
- Obtain independent read-only review before committing.

## Rollback

Revert the implementation commit. The change has no schema, data, protocol, or configuration
migration.
