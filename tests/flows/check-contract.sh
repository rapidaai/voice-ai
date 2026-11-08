#!/usr/bin/env bash
set -Eeuo pipefail

flow_root=${1:-"$(cd "$(dirname "$0")" && pwd)"}
manifest="$flow_root/manifest.txt"
supported_clients='nodejs react rest'
journeys=()
fixture_ids=()
fixture_id_owners=()

fail() {
  printf 'flow contract: %s\n' "$1" >&2
  exit 1
}

logical_commands() {
  awk '
    function emit_command(complete_command) {
      complete_command = command
      sub(/^[[:space:]]+/, "", complete_command)
      sub(/[[:space:]]+$/, "", complete_command)
      if (complete_command != "") print complete_command
      command = ""
    }
    function split_commands(line, character_index, character, next_character, previous_character,
                            escaped, single_quoted, double_quoted) {
      escaped = single_quoted = double_quoted = 0
      for (character_index = 1; character_index <= length(line); character_index++) {
        character = substr(line, character_index, 1)
        next_character = substr(line, character_index + 1, 1)
        previous_character = substr(line, character_index - 1, 1)
        if (escaped) {
          command = command character
          escaped = 0
          continue
        }
        if (single_quoted) {
          command = command character
          if (character == "\047") single_quoted = 0
          continue
        }
        if (double_quoted) {
          command = command character
          if (character == "\\") escaped = 1
          else if (character == "\"") double_quoted = 0
          continue
        }
        if (character == "\\") {
          command = command character
          escaped = 1
          continue
        }
        if (character == "\047") {
          command = command character
          single_quoted = 1
          continue
        }
        if (character == "\"") {
          command = command character
          double_quoted = 1
          continue
        }
        if (character == "#" && (character_index == 1 || previous_character ~ /[[:space:]]/)) break
        if (character == ";") {
          emit_command()
          continue
        }
        if (character == "&" && previous_character != ">") {
          emit_command()
          if (next_character == "&") character_index++
          continue
        }
        if (character == "|") {
          emit_command()
          if (next_character == "|") character_index++
          continue
        }
        command = command character
      }
      emit_command()
    }
    {
      line = $0
      if (line ~ /\\[[:space:]]*$/) {
        sub(/\\[[:space:]]*$/, "", line)
        continued = continued line " "
        next
      }
      split_commands(continued line)
      continued = ""
    }
    END { if (continued != "") split_commands(continued) }
  ' "$1"
}

temporary_creation_line() {
  awk '
    NR == FNR {
      if (!inside_function && $0 ~ /^[[:space:]]*[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(\)[[:space:]]*\{/) {
        function_name = $0
        sub(/^[[:space:]]*/, "", function_name)
        sub(/[[:space:]]*\(\).*/, "", function_name)
        if ($0 ~ /mktemp -d/) creates_temporary_directory[function_name] = 1
        inside_function = ($0 !~ /}[[:space:]]*$/)
        next
      }
      if (inside_function && $0 ~ /mktemp -d/) creates_temporary_directory[function_name] = 1
      if (inside_function && $0 ~ /^[[:space:]]*}[[:space:]]*$/) inside_function = 0
      next
    }
    {
      if (!inside_function && $0 ~ /^[[:space:]]*[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\(\)[[:space:]]*\{/) {
        inside_function = ($0 !~ /}[[:space:]]*$/)
        next
      }
      if (inside_function) {
        if ($0 ~ /^[[:space:]]*}[[:space:]]*$/) inside_function = 0
        next
      }
      if ($0 ~ /mktemp -d/) {
        print FNR
        exit
      }
      invocation = $0
      sub(/^[[:space:]]*/, "", invocation)
      sub(/[[:space:];|&].*$/, "", invocation)
      if (creates_temporary_directory[invocation]) {
        print FNR
        exit
      }
    }
  ' "$1" "$1"
}

[[ -f "$manifest" ]] || fail "missing manifest: $manifest"

while IFS= read -r journey || [[ -n "$journey" ]]; do
  [[ -n "$journey" ]] || fail 'manifest contains an empty line'
  [[ "$journey" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ ]] || fail "invalid journey name: $journey"

  for listed_journey in "${journeys[@]}"; do
    [[ "$listed_journey" != "$journey" ]] || fail "duplicate journey in manifest: $journey"
  done
  journeys+=("$journey")

  journey_directory="$flow_root/$journey"
  journey_runner="$journey_directory/run.sh"
  [[ -d "$journey_directory" ]] || fail "missing journey directory: $journey"
  [[ -x "$journey_runner" ]] || fail "journey runner is missing or not executable: $journey/run.sh"
  grep -Fqx 'set -eu' "$journey_runner" || fail "$journey/run.sh must use set -eu"
  # shellcheck disable=SC2016 # The runner must contain this literal variable reference.
  grep -Fqx '. "$script_directory/../report.sh"' "$journey_runner" || fail "$journey/run.sh must source report.sh"

  declaration_count=$(grep -Ec "^run_flow_clients \"\\\$script_directory\" '[^']+'( (nodejs|react|rest))+$" "$journey_runner" || true)
  [[ "$declaration_count" -eq 1 ]] || fail "$journey/run.sh must contain one standard run_flow_clients declaration"
  declaration=$(grep -E "^run_flow_clients \"\\\$script_directory\" '[^']+'( (nodejs|react|rest))+$" "$journey_runner")
  clients=$(sed -E "s/^run_flow_clients \"\\\$script_directory\" '[^']+' //" <<<"$declaration")
  case " $clients " in
    *' react '*) ;;
    *) fail "$journey/run.sh must declare react coverage" ;;
  esac

  for client in $clients; do
    case " $supported_clients " in
      *" $client "*) ;;
      *) fail "$journey/run.sh declares unsupported client: $client" ;;
    esac

    client_runner="$journey_directory/$client/run.sh"
    [[ -x "$client_runner" ]] || fail "client runner is missing or not executable: $journey/$client/run.sh"
    grep -Fqx 'set -eu' "$client_runner" || fail "$journey/$client/run.sh must use set -eu"
    # shellcheck disable=SC2016 # The runner must contain this literal variable reference.
    grep -Fqx '. "$script_directory/../../process-lifecycle.sh"' "$client_runner" || \
      fail "$journey/$client/run.sh must source process-lifecycle.sh"
    grep -Eq '^register_flow_cleanup [a-z_]+$' "$client_runner" || \
      fail "$journey/$client/run.sh must register cleanup with the process lifecycle"
    temporary_line=$(temporary_creation_line "$client_runner")
    if [[ -n "$temporary_line" ]]; then
      cleanup_line=$(grep -n '^register_flow_cleanup ' "$client_runner" | head -1 | cut -d: -f1)
      [[ "$cleanup_line" -lt "$temporary_line" ]] || \
        fail "$journey/$client/run.sh must register cleanup before creating temporary files"
    fi

    if [[ "$client" == 'rest' ]]; then
      operation_count=0
      while IFS= read -r operation; do
        operation_count=$((operation_count + 1))
        [[ "$operation" =~ --max-time[[:space:]]+0*[1-9][0-9]*([[:space:]]|\\|$) ]] || \
          fail "$journey/$client/run.sh must bound every REST call with a positive --max-time"
      done < <(logical_commands "$client_runner" | grep -E '(^|[^[:alnum:]_])curl([[:space:]]|$)' || true)
      [[ "$operation_count" -gt 0 ]] || fail "$journey/$client/run.sh must contain a REST operation"
    else
      operation_count=0
      while IFS= read -r operation; do
        operation_count=$((operation_count + 1))
        [[ "$operation" =~ timeout[[:space:]]+0*[1-9][0-9]*s[[:space:]]+node[[:space:]] ]] || \
          fail "$journey/$client/run.sh must bound every SDK call with a positive timeout"
      done < <(logical_commands "$client_runner" | grep -E '(^|[^[:alnum:]_])node([[:space:]]|$)' || true)
      [[ "$operation_count" -gt 0 ]] || fail "$journey/$client/run.sh must contain an SDK operation"
    fi

    if grep -qE '(^|[[:space:]])psql[[:space:]]' "$client_runner"; then
      while IFS= read -r psql_line; do
        # shellcheck disable=SC2016 # The runner must contain this literal environment fallback.
        [[ "$psql_line" == *'${FLOW_POSTGRES_HOST:-postgres}'* ]] || fail "$journey/$client/run.sh must use FLOW_POSTGRES_HOST for psql"
      done < <(grep -E '(^|[[:space:]])psql[[:space:]]' "$client_runner")
    fi

    while IFS= read -r javascript_file; do
      node --check "$javascript_file" >/dev/null || \
        fail "$journey/$client/$(basename "$javascript_file") contains invalid JavaScript"
      if [[ "$client" == 'react' ]] && grep -Fq 'ConnectionConfig' "$javascript_file" && ! grep -Fq 'FLOW_SDK_DEBUG' "$javascript_file"; then
        fail "$journey/$client/$(basename "$javascript_file") must make SDK debug logging opt-in with FLOW_SDK_DEBUG"
      fi
    done < <(find "$journey_directory/$client" -maxdepth 1 -type f -name '*.js' -print | sort)

    while IFS=: read -r assignment_line assignment; do
      fixture_id=$(sed -E "s/.*=['\"]([0-9]+)['\"].*/\1/" <<<"$assignment")
      fixture_owner="$journey/$client/run.sh:$assignment_line"
      for fixture_index in "${!fixture_ids[@]}"; do
        if [[ "${fixture_ids[$fixture_index]}" == "$fixture_id" ]]; then
          fail "fixture identifier $fixture_id is reused by ${fixture_id_owners[$fixture_index]} and $fixture_owner"
        fi
      done
      fixture_ids+=("$fixture_id")
      fixture_id_owners+=("$fixture_owner")
    done < <(grep -En "^[[:space:]]*(export[[:space:]]+)?(readonly[[:space:]]+)?[A-Za-z0-9_]*(ID|_id)=['\"][0-9]+['\"]$" "$client_runner" || true)

    while IFS=: read -r assignment_line assignment; do
      fixture_email=$(sed -E "s/.*=['\"]([^'\"]+)['\"].*/\1/" <<<"$assignment")
      [[ "$fixture_email" != *'$'* ]] || continue
      [[ "$fixture_email" == *@*.invalid ]] || \
        fail "$journey/$client/run.sh:$assignment_line must use a reserved .invalid email address"
    done < <(grep -En "^[[:space:]]*(export[[:space:]]+)?(readonly[[:space:]]+)?[A-Za-z0-9_]*(EMAIL|email)=['\"][^'\"]+['\"]$" "$client_runner" || true)

    if grep -Fq '../../project-fixture.sh' "$client_runner"; then
      seed_line=$(grep -n '^seed_project_fixture$' "$client_runner" | head -1 | cut -d: -f1)
      [[ -n "$seed_line" ]] || fail "$journey/$client/run.sh must seed the shared project fixture"
      if [[ "$client" == 'rest' ]]; then
        operation_line=$(grep -nE '^run_flow( |$)|^run_[a-z_]+_checks$' "$client_runner" | head -1 | cut -d: -f1)
      else
        operation_line=$(grep -nE '^run_flow_process timeout [0-9]+s node ' "$client_runner" | head -1 | cut -d: -f1)
      fi
      [[ -n "$operation_line" && "$seed_line" -lt "$operation_line" ]] || \
        fail "$journey/$client/run.sh must seed before exercising the public client"
      verify_line=$(grep -nE '^verify_[a-z_]+_fixture$' "$client_runner" | head -1 | cut -d: -f1 || true)
      if [[ -n "$verify_line" && "$operation_line" -ge "$verify_line" ]]; then
        fail "$journey/$client/run.sh must verify persistent state after exercising the public client"
      fi
    fi
  done

  for client_directory in "$journey_directory"/*/; do
    [[ -d "$client_directory" ]] || continue
    client=$(basename "$client_directory")
    case " $clients " in
      *" $client "*) ;;
      *) fail "$journey contains undeclared client directory: $client" ;;
    esac
  done
done < "$manifest"

[[ "${#journeys[@]}" -gt 0 ]] || fail 'manifest contains no journeys'

for journey_directory in "$flow_root"/*/; do
  [[ -d "$journey_directory" ]] || continue
  journey=$(basename "$journey_directory")
  is_listed=false
  for listed_journey in "${journeys[@]}"; do
    if [[ "$listed_journey" == "$journey" ]]; then
      is_listed=true
      break
    fi
  done
  [[ "$is_listed" == true ]] || fail "journey directory is missing from manifest: $journey"
done

printf 'Flow contract passed for %d journeys\n' "${#journeys[@]}"
