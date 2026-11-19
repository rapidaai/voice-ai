# RFC 0083: Client-Owned Service Issuer

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-14
- Supersedes: RFC 0082 temporary retention of `AppConfig.ServiceAssertionIssuer` only

## Context

`AppConfig.ServiceAssertionIssuer` is a one-line forwarding method. Its sole production caller is
`pkg/clients.NewInternalClient`, which already owns the outbound destination and issuer error
lifecycle. The repository owner requires the temporary config method to be removed now that its
derivation dependencies are clean.

## Decision

Delete `AppConfig.ServiceAssertionIssuer`. `pkg/clients.NewInternalClient` constructs its
destination-bound issuer directly with
`jwt.NewIssuer(destination, cfg.ServiceAuthenticationKey())` and continues storing the returned
error for request-time propagation.

## Contracts and Ownership

- `AppConfig` owns only secret storage and deterministic key derivation.
- `pkg/clients.NewInternalClient` owns outbound destination-bound issuer construction.
- `pkg/authentication/jwt.NewIssuer` retains validation and error ownership.
- Issued JWT bytes, validation behavior, error types, and request behavior remain unchanged.

## Alternatives Rejected

- Keep the forwarding method: rejected because it gives config ownership of a client resource.
- Make JWT issuer construction infallible: rejected because no JWT contract change is required.
- Construct an issuer per request: rejected because the immutable issuer belongs to the client
  lifetime.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | The exported config convenience method is intentionally removed. Repository search finds one production caller, which moves atomically. Runtime behavior is unchanged. |
| Failure and recovery | Invalid destinations continue producing `JWTIssuerInvalidError`, stored and returned by the client. Rollback restores the forwarding method. |
| Security and privacy | JWT validation, key bytes, claims, lifetime, and secret handling are unchanged. |
| Data and migration | N/A. No persisted data or schema changes. |
| Concurrency and resources | One immutable issuer remains owned per internal client. No lifetime change. |
| Operations and observability | N/A. No logs, metrics, or deployment behavior change. |

## Rollout and Rollback

Deploy with the authentication cutover. No mixed-version protocol risk exists because runtime JWT
behavior is unchanged. Rollback restores the config forwarding method and its single caller.

## Verification

```bash
! rg -n 'func \(cfg \*AppConfig\) ServiceAssertionIssuer|\.ServiceAssertionIssuer\(' --glob '*.go' .
go test -count=1 ./config ./pkg/clients/...
go test -race -count=1 ./config ./pkg/clients
git diff --check
just agent-finalize "config/config.go,config/config_test.go,pkg/clients/internal_client.go,pkg/clients/internal_client_test.go,rfcs/0083-client-owned-service-issuer.md,plans/0083-client-owned-service-issuer.md"
```

## Review and Approval

- Challenger: Independent GPT-6.1 Sol and GPT-6 Astra review
- Outcome: Approved
- Resolved findings: Required invalid-issuer coverage for both direct-service and delegated-request
  paths; narrowed the cleanup scan to the removed config method and calls so it does not match an
  unrelated SIP option field.
- Open questions: None
- Approval reference: Repository owner explicitly directed removal from config and requested the correct owner on 2026-11-14.
