# Agent Tooling Lessons

This ledger records recurring, evidence-backed defects in the repository development workflow.
Each lesson must identify the failure, the durable rule, and its executable guard. Do not add
speculative preferences.

| ID | Observed failure | Durable rule | Guard or regression test |
| --- | --- | --- | --- |
| ATL-001 | A publishing command nested in a shell could avoid direct command matching. | A security-sensitive command that cannot be parsed safely fails closed. | `tests/agent_tooling/test_pr_gate.py` exercises nested publishing commands. |
| ATL-002 | Generated policy files under `ui/src/` selected the full UI test suite despite changing no UI source. | Test selection uses source-file types, not directory membership alone. | `tests/agent_tooling/test_policy_tools.py` covers policy and UI source paths. |
| ATL-003 | Documentation placed inside a submodule boundary was invisible to the parent repository. | New repository policy documents remain in a path owned by the parent repository. | `bin/check-agent-docs` validates the root documentation set. |
| ATL-004 | Hook behavior without regression coverage can drift silently. | Every deny, allow, injection, and publication path added to a hook includes a focused test. | `just test-agent-hooks` runs the repository hook suite. |
| ATL-005 | The UI provider suite passed but remained in interactive mode until the finalizer timed out. | Finalizer-owned UI tests set CI mode and disable watch mode explicitly. | `tests/agent_tooling/test_policy_tools.py` locks the non-interactive command and environment. |
| ATL-006 | Synthetic repositories inherited the caller hook's Git state, which redirected fixtures into the real repository. | Validators clear repository-local Git environment variables before creating fixtures, and fixture commits use Git plumbing. | The pre-push `agent-pr-ready` gate runs `bin/validate-development-process` in its hook environment. |

Add a lesson only after a concrete defect or repeated review finding. Prefer extending an existing
lesson when the invariant is unchanged.
