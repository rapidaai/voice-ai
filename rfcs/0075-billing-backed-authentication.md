# RFC 0075: Billing-Backed Authentication

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-13
- Supersedes: RFC 0074 billing-source deferral only

## Context

RFC 0074 makes BillingPlan an optional Organization-owned part of the unified Authentication
aggregate, but intentionally defers its persistence source and runtime population. The historical
`feat/billing-foundation` branch contains the intended plan, quota, and subscription concepts, but
its implementation depends on the removed legacy authentication model and obsolete migration
numbers.

The Web API currently has no authoritative billing-plan tables or service. Copying the historical
commit would restore legacy principals, controller-owned authorization, unrelated UI code, and old
migrations. A focused persistence port is required before Authentication can expose verified
billing facts.

## Decision

Add Web API-owned `billing_plans`, `billing_plan_quotas`, and `billing_subscriptions` tables after
migration 000017. Plans are platform records, quotas belong to one plan, and each Organization may
have at most one subscription to one plan. Migration 000019 seeds the Free, Pro, and Enterprise
catalogs and assigns the Free plan to existing Organizations. Migration-owned audit rows use the
existing `unknown` actor with a null actor ID.

`BillingPlanQuota.quotaLimit` accepts `-1` as the sole unlimited sentinel and otherwise requires a
nonnegative value. This supersedes RFC 0074's nonnegative-only quota rule. Values below `-1` remain
invalid in both the database and immutable aggregate.

Add matching Web API entities and a billing service that reads active Organization subscriptions,
plans, and ordered quotas in one persistence operation. It does not authorize routes, expose a
billing controller, mutate subscriptions, or own Authentication construction.

The User service remains the owner of the authenticated User membership graph. When it constructs
the RFC 0074 Authentication aggregate, it obtains the Organizations' subscriptions with one batch
read and maps each active plan and its ordered quotas into that Organization. A missing
subscription leaves BillingPlan absent. Database failures and an active subscription referencing a
missing or inactive plan fail Authentication construction rather than silently returning incomplete
billing facts.

Billing is loaded only while producing the full Authentication aggregate. Principal resolvers and
route middleware do not read billing. Login, registration, and social authentication build a direct
User Principal from the exact verified session ID and the same canonical role graph. Authorize
preserves the middleware Principal and loads the exact session named by its proof reference. The
new aggregate constructor remains responsible for detecting any mismatch between the Principal's
authority snapshot and the freshly loaded role graph.

Every successful gRPC `AuthenticateResponse` producer populates field 5 with the unified
Authentication aggregate and its exact issued token. Deprecated field 3 remains populated from the
same canonical snapshot during this compatibility phase. Gin legacy JSON remains unchanged.
Controllers only map the service result to protobuf; credential verification, membership loading,
billing loading, Principal construction, and aggregate validation remain in the User service.

Historical billing controllers, UI routes, legacy Principal fields, `types.Principle` billing
accessors, and subscription mutation are excluded.

## Contracts and Ownership

- Web API schema owns billing plans, quotas, and Organization subscriptions.
- The billing service owns billing persistence reads and returns domain entities only.
- The User service owns User Authentication construction and combines canonical memberships with
  billing facts.
- `pkg/authentication.Authentication` remains the immutable validated security contract.
- Organization owns BillingPlan. Projects and roles never copy billing facts.
- The controller owns only protobuf presentation and preserves the deprecated response during the
  compatibility window.
- The current Web API permits one Organization membership per User. The aggregate remains capable
  of representing multiple Organizations, but this change does not remove that existing service
  restriction.

## Alternatives Rejected

- Cherry-pick the historical commit: rejected because it restores legacy authentication and
  unrelated UI code and uses obsolete migration numbers.
- Put billing reads in the controller: rejected because it would let presentation code assemble a
  security contract and decide failure behavior.
- Store BillingPlan directly on Organization: rejected because a subscription has its own status,
  billing interval, and period lifecycle.
- Ignore billing read failures: rejected because callers could interpret an incomplete aggregate as
  a real no-plan state.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | Deprecated authentication data remains available while field 5 starts carrying the new aggregate. |
| Failure and recovery | Authentication fails if its required billing query fails. An Organization without a subscription remains valid with no BillingPlan. |
| Security and privacy | Billing is loaded only for Organizations already present in the canonical membership graph. No payment secrets are stored in Authentication. |
| Data and migration | Three tables and seeded catalog data are added. Existing Organizations receive a Free subscription. New Organizations may have no plan until a later accepted atomic provisioning change. |
| Concurrency and resources | One set-based billing query avoids per-Organization reads. Unique constraints prevent duplicate active Organization subscriptions. |
| Operations and observability | Migration failure stops rollout. Authentication errors retain the underlying persistence failure for the owning boundary. |

## Rollout and Rollback

Deploy migrations 000018 and 000019 before the Web API binary. Stop rollout if catalog seeding,
Organization backfill, aggregate tests, or login compatibility tests fail.

Rollback the binary first. Migration 000019 marks its backfilled subscriptions in metadata and
removes only unchanged migration-owned Free subscriptions before removing seeded catalog rows. Its
down migration fails closed when migration rows changed or later subscription data references the
catalog. Migration 000018 fails closed unless every billing table is empty, then drops the tables
in dependency order. Each migration is transactional. Operators retain a database backup before
schema rollback.

## Verification

- Billing entity and service tests cover active subscription loading, ordered quotas, a missing
  subscription, and database failure.
- Authentication tests cover BillingPlan population, absent plans, immutable output, and billing
  query failure.
- Authentication endpoint tests cover field 5 and the deprecated compatibility field for password,
  registration, Authorize, and social success paths.
- Migration verification applies 000018 and 000019 after 000017, checks seeded and backfilled data,
  and exercises rollback on unchanged seed data.
- Run `go test ./api/web-api/internal/entity ./api/web-api/internal/service/billing ./api/web-api/internal/service/user ./api/web-api/api ./pkg/authentication`.
- Run `go test -race ./api/web-api/internal/service/billing ./api/web-api/internal/service/user ./pkg/authentication`.
- Run `just agent-finalize "api/web-api/internal/entity,api/web-api/internal/service,api/web-api/migrations,api/web-api/api,pkg/authentication,rfcs"`.

## Review and Approval

- Challenger: Independent schema and Authentication ownership reviews
- Outcome: Approved
- Resolved findings: Defined the unlimited sentinel, current and future Organization behavior,
  exact-session ownership, batch loading, corrupt-reference failure, and guarded rollback.
- Open questions: None
- Approval reference: User instruction on 2026-11-13 to port billing persistence and integrate it with the new Authentication aggregate.
