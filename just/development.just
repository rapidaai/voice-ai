# Validate development process rules.
validate-development-process:
    bash bin/validate-development-process

# Validate agent tooling rules.
validate-agent-tooling:
    bash bin/validate-agent-tooling

# Validate RFC structure and metadata.
validate-rfc-layout:
    python3 bin/validate-rfc-layout

# Validate all development tooling.
validate-development-toolkit: validate-development-process validate-agent-tooling validate-rfc-layout

# Run scoped repository finalization checks.
agent-finalize changed_files=env_var_or_default("CHANGED_FILES", "") agent_root=env_var_or_default("AGENT_ROOT", ".codex"):
    @test -n "{{changed_files}}" || { echo "Usage: just agent-finalize 'path/a.go,path/a_test.go' [agent-root]" >&2; exit 2; }
    AGENT_ROOT="{{agent_root}}" CHANGED_FILES="{{changed_files}}" bash bin/agent-finalize

# Sign an accepted implementation plan.
sign-approved-plan run_id=env_var_or_default("RUN_ID", "") plan=env_var_or_default("PLAN", ""):
    @test -n "{{run_id}}" -a -n "{{plan}}" || { echo "Usage: DEVELOPMENT_GATE_KEY=... just sign-approved-plan '<run-id>' 'rfcs/NNNN-name/jsons/plan.json'" >&2; exit 2; }
    bash bin/sign-approved-plan "{{run_id}}" "{{plan}}"

# Start a governed Orca development run.
orca-development-run objective=env_var_or_default("OBJECTIVE", "") rfc=env_var_or_default("RFC", "") agent=env_var_or_default("AGENT", "codex"):
    @test -n "{{objective}}" -a -n "{{rfc}}" || { echo "Usage: just orca-development-run '<objective>' 'rfcs/NNNN-name.md' [agent]" >&2; exit 2; }
    bash bin/orca-development-run "{{objective}}" "{{agent}}" "{{rfc}}"

# Release an abandoned governed RFC run.
orca-rfc-release rfc=env_var_or_default("RFC", ""):
    @test -n "{{rfc}}" || { echo "Usage: just orca-rfc-release 'rfcs/NNNN-name.md'" >&2; exit 2; }
    bash bin/orca-rfc-release "{{rfc}}"

# Create the RFC confirmation task and gate.
orca-confirm-rfc-create run_id=env_var_or_default("RUN_ID", "") challenge_task_id=env_var_or_default("CHALLENGE_TASK_ID", "") challenge_receipt=env_var_or_default("CHALLENGE_RECEIPT", "") rfc=env_var_or_default("RFC", ""):
    @test -n "{{run_id}}" -a -n "{{challenge_task_id}}" -a -n "{{challenge_receipt}}" -a -n "{{rfc}}" || { echo "Usage: just orca-confirm-rfc-create '<run>' '<challenge-task>' '<challenge-receipt>' '<rfc>'" >&2; exit 2; }
    python3 bin/orca-confirm-rfc create --run "{{run_id}}" --challenge-task "{{challenge_task_id}}" --challenge-receipt "{{challenge_receipt}}" --rfc "{{rfc}}"

# Collect and verify the RFC confirmation receipt.
orca-confirm-rfc-collect run_id=env_var_or_default("RUN_ID", "") task_id=env_var_or_default("TASK_ID", "") challenge_task_id=env_var_or_default("CHALLENGE_TASK_ID", "") challenge_receipt=env_var_or_default("CHALLENGE_RECEIPT", "") gate_id=env_var_or_default("GATE_ID", "") rfc=env_var_or_default("RFC", "") receipt=env_var_or_default("RECEIPT", ""):
    #!/usr/bin/env bash
    test -n "{{run_id}}" -a -n "{{task_id}}" -a -n "{{challenge_task_id}}" -a -n "{{challenge_receipt}}" -a -n "{{gate_id}}" -a -n "{{rfc}}" || {
      echo "Usage: just orca-confirm-rfc-collect '<run>' '<task>' '<challenge-task>' '<challenge-receipt>' '<gate>' '<rfc>' [receipt]" >&2
      exit 2
    }
    args=(collect --run "{{run_id}}" --task "{{task_id}}" --challenge-task "{{challenge_task_id}}" --challenge-receipt "{{challenge_receipt}}" --gate "{{gate_id}}" --rfc "{{rfc}}")
    if [[ -n "{{receipt}}" ]]; then
      args+=(--output "{{receipt}}")
    fi
    python3 bin/orca-confirm-rfc "${args[@]}"

# Render the Orca development panel.
orca-panel input=env_var_or_default("PANEL_INPUT", ".codex/orchestrator/examples/lifecycle-input.json") output=env_var_or_default("PANEL_OUTPUT", ".cache/orca-development-panel.html"):
    python3 bin/orca-development-panel --input "{{input}}" --output "{{output}}"

# Render and open the Orca development panel.
orca-panel-open input=env_var_or_default("PANEL_INPUT", ".codex/orchestrator/examples/lifecycle-input.json") output=env_var_or_default("PANEL_OUTPUT", ".cache/orca-development-panel.html"):
    python3 bin/orca-development-panel --input "{{input}}" --output "{{output}}" --open
