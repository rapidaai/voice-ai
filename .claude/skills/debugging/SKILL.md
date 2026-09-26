---
name: debugging
description: Diagnose regressions, failing tests, runtime errors, and unexpected behavior before applying a fix. Use when the cause is unknown or when a proposed fix has not been tied to a reproducible failure.
---

# Debugging

## Mission

Produce a reproducible explanation of the failure, then prove the smallest correct fix with a regression test.

## Respect the request

If the user asked only for diagnosis, stop after reporting the supported root cause and fix options. Do not edit production code without authorization.

## Debugging sequence

1. Record expected behavior, actual behavior, environment, and the smallest known reproduction.
2. Reproduce the failure with the narrowest command that exercises the real path.
3. Trace ownership and state from the first incorrect observation backward to the responsible boundary.
4. Form one falsifiable hypothesis. State what evidence would confirm or reject it.
5. Run the smallest safe check that discriminates that hypothesis.
6. Repeat one hypothesis at a time until the root cause explains all observed evidence.
7. Add a regression test and watch it fail for the expected reason before changing behavior.
8. Apply the smallest fix at the owning layer, then run focused and fallback-path verification.

## Failure analysis

Check invalid input, cancellation, timeout, retry, partial success, cleanup, goroutine lifetime, packet ordering, and provider fallback when relevant. Do not patch a downstream symptom while leaving the owning transition incorrect.

## Evidence log

Maintain a compact record:

- reproduction command and result
- hypothesis and discriminating check
- root cause with `file:line`
- regression test and its pre-fix failure
- changed owner and why it is the correct layer
- focused and broader verification results

## Escalation

Return to `development-lifecycle` when the root cause expands scope or reveals a Governed trigger. For voice integration failures, activate the matching domain skill before editing.

## Governed lifecycle

Debugging evidence may amend a Governed plan, but any changed RFC bytes or approved scope must pass the required challenge and confirmation again. This skill must not self-approve that amendment.
