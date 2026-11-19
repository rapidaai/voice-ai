# RFC 0048: Remove User Role Migration Tests

- Status: Accepted
- Owner: Access Management
- Approved by: Repository owner
- Date: 2026-11-10
- Supersedes: RFC 0047 automated migration-verification requirements only

## Context

RFC 0047 introduced migration 000017 and required persistent automated coverage in
`bin/verify-role-catalog-migration.sh`. The repository owner has directed that migration tests not
be added. The migration SQL and its deployment ordering remain approved and unchanged.

## Decision

Remove only the migration-17-specific setup, assertions, failure cases, down and re-up execution,
and injected rollback cases from `bin/verify-role-catalog-migration.sh`. Do not modify existing
role-catalog or project-token verification that predates migration 000017.

Migration 000017 remains transactional and fail-closed as accepted in RFC 0047. Its up and down SQL
remain unchanged. No replacement migration test file or test utility is introduced. Development
evidence already obtained from applying up, down, re-up, invalid-value, and injected-failure paths
is retained in the task record rather than committed as repository automation.

## Contracts and Ownership

- Migration 000017 continues to own the exact role mappings, witnesses, foreign keys, uniqueness,
  rollback guards, and maintenance-window requirement accepted in RFC 0047.
- `bin/verify-role-catalog-migration.sh` continues to own its pre-existing catalog and project-token
  verification only.
- No production schema, data, authentication, authorization, API, or UI contract changes.

## Alternatives Rejected

- Keep migration-17 coverage in the shared verifier: rejected by repository-owner direction.
- Move the same coverage into another test file: rejected because it would preserve the migration
  tests under a different name.
- Remove the existing catalog and project-token verifier: rejected because that work predates this
  migration and is outside the requested scope.

## Consequences and Risks

| Area | Impact or reason for N/A |
| --- | --- |
| Compatibility | N/A. Production migration SQL and runtime contracts are unchanged. |
| Failure and recovery | The SQL retains transactional failure and guarded rollback, but future regressions in migration 000017 will not have persistent automated coverage. |
| Security and privacy | Authorization constraints remain unchanged. The loss of regression coverage can delay detection of an accidental constraint change. |
| Data and migration | Migration behavior is unchanged. Verification evidence is point-in-time rather than continuously executable. |
| Concurrency and resources | N/A. Test-only shell logic is removed; migration locks remain unchanged. |
| Operations and observability | Operators still follow RFC 0047 preflight, maintenance-window, and post-migration checks. CI will not exercise migration 000017. |

## Rollout and Rollback

Remove the migration-17 verifier additions before delivery. The removal is test-only and needs no
runtime rollout. Restoring automated coverage requires repository-owner approval and a later change.

## Verification

Acceptance requires no migration-17 fixtures, assertions, execution, or injected failures remain
in `bin/verify-role-catalog-migration.sh`; migration 000017 SQL retains the independently reviewed
SHA-256 values below; the retained catalog and project-token verifier passes; and the shell script
remains syntactically valid.

Exact commands:

```text
bash -n bin/verify-role-catalog-migration.sh
rg -n "000017|user_role_unknown|user_role_rollback|9201|9301|9401|9501" bin/verify-role-catalog-migration.sh
shasum -a 256 api/web-api/migrations/000017_migrate_user_role_assignments.up.sql api/web-api/migrations/000017_migrate_user_role_assignments.down.sql
bash bin/verify-role-catalog-migration.sh
git diff --check -- bin/verify-role-catalog-migration.sh rfcs/0048-remove-user-role-migration-tests.md rfcs/README.md
just validate-rfc-layout
just agent-finalize "bin/verify-role-catalog-migration.sh,rfcs/0048-remove-user-role-migration-tests.md,rfcs/README.md"
```

The `rg` command must return no matches. The SHA-256 output must remain:

```text
84ef9a15fb7948193fece4f59d4deb79f935bcb87f138ad096c47e4cad60b741  api/web-api/migrations/000017_migrate_user_role_assignments.up.sql
3e7fc2295e4b13d572bc207c77a93203a19fcec8bab534b946feefe49617fbd5  api/web-api/migrations/000017_migrate_user_role_assignments.down.sql
```

## Review and Approval

- Challenger: Independent Codex task `/root/user_role_migration_challenge`
- Outcome: Approved
- Resolved findings: Migration SQL is pinned to independently reviewed SHA-256 values, and retained catalog and project-token verification must pass after removal.
- Open questions: None
- Approval reference: Repository owner instruction on 2026-11-10: "doe not write test for migration".
