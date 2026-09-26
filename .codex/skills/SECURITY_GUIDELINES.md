# Codex Skills Security Guidelines

These rules apply to all files under `.codex/skills/`.

## 1) No secrets in skills

Never commit secrets in:
- `SKILL.md`
- `examples/`
- `references/`
- `agents/openai.yaml`
- `scripts/`

Disallowed secret patterns include API keys, bearer tokens, passwords, long-lived credentials, and private endpoints containing credentials.

## 2) No unsafe instructions

Do not add instructions that ask agents to:
- bypass safety controls
- hide actions from users
- exfiltrate data
- execute destructive commands without explicit user intent

## 3) Principle of least scope

Skill instructions and validators must constrain edits to:
- provider-specific folder(s)
- required factory switch file(s)
- minimal shared contract files

Avoid broad wildcard permissions when a provider-specific path is possible.

## 4) Validation is mandatory

Before merging skill updates:

1. run basic validation
2. run strict validation with `--provider` where applicable
3. review failed-path behavior and boundary checks

## 5) Public contribution review checklist

- Verify examples do not contain real credentials.
- Verify commands are non-destructive by default.
- Verify out-of-scope boundaries are explicit.
- Verify EOS and VAD separation remains intact.
- Verify telemetry guidance avoids sensitive payload capture.

## 6) Incident response for leaked secrets

If a secret is committed:

1. revoke/rotate immediately
2. remove the secret from source and history as per repo policy
3. update validator patterns if needed
4. document the fix in PR notes

## 7) Ownership

Security review is required for any skill change that:
- broadens validation scope
- introduces new shell/network behavior
- changes credential handling guidance
