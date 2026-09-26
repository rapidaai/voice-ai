---
name: preparing-delivery
description: Prepare a verified change for commit, push, or pull request after the user explicitly requests that delivery action. Use for conventional commit preparation, PR descriptions, final evidence checks, and delivery readiness.
---

# Preparing Delivery

## Mission

Deliver exactly the reviewed change with accurate verification, operational, and rollback information.

## Authorization boundary

Do not commit, push, create or edit a pull request, merge, tag, deploy, or release unless the user explicitly requested that action. Authorization for one action does not imply another.

## Preflight

1. Confirm the branch, base, status, and complete diff.
2. Confirm changed files match the approved scope and contain no unrelated work.
3. Confirm required finalization and review are complete.
4. Confirm no unresolved Critical or Major finding remains.
5. Confirm generated artifacts came from their source and generator.
6. Re-run any check invalidated by changes made after review.
7. Confirm any configured independent-review panel still matches the final diff digest.

## Commit preparation

- Use the Conventional Commit rules in `AGENTS.md`.
- Keep one coherent change per commit.
- Describe the behavioral outcome, not the editing activity.
- Never bypass hooks or rewrite unrelated history.

## Pull request description

Start from `.github/PULL_REQUEST_TEMPLATE.md`, write the completed body to a repository-local file,
and run `just agent-pr-ready <base>` before creating or marking the pull request ready. The local
Claude PR hook and the `Agent Policy` workflow enforce the same policy.

Include:

- problem and impact
- behavior before and after
- implementation scope and explicit exclusions
- contract, compatibility, security, data, and operational impact
- exact verification commands and results
- independent review decision and resolved findings
- rollout, rollback, or disablement when applicable
- screenshots or UI evidence when applicable

Do not claim a check passed unless its result is available. Distinguish verified behavior from assumptions and environmental limitations.

Do not use bypass flags, skip variables, or inline pull request bodies. Content sent through a
repository-owned external-review command must pass `bin/agent-egress` first.

## Final check

After any delivery mutation, confirm the resulting branch or PR points to the reviewed bytes. Never merge, tag, deploy, or release as an implied follow-up.

## Governed lifecycle

Governed delivery requires the accepted RFC, exact-digest confirmation, complete lifecycle evidence, passing verification, and approved independent review. This skill must not self-approve missing evidence.
