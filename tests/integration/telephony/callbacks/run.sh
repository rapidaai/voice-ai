#!/bin/sh
set -eu

readonly assistant_api_url="${ASSISTANT_API_URL:-http://assistant-api:9007}"
readonly postgres_host="${POSTGRES_HOST:-postgres}"
readonly postgres_user="${POSTGRES_USER:-rapida_user}"
readonly postgres_database="${POSTGRES_DATABASE:-assistant_db}"
readonly requested_provider="${1:-all}"
readonly conversation_ids='7100001,7100002,7100003,7100004,7100005,7100006,7100007,7100008'

temporary_directory=$(mktemp -d)
failures=0

query() {
  psql -v ON_ERROR_STOP=1 -h "$postgres_host" -U "$postgres_user" -d "$postgres_database" -Atc "$1"
}

cleanup() {
  query "DELETE FROM assistant_conversation_metadata WHERE assistant_conversation_id IN ($conversation_ids)" >/dev/null || true
  query "DELETE FROM assistant_conversation_metrics WHERE assistant_conversation_id IN ($conversation_ids)" >/dev/null || true
  query "DELETE FROM call_contexts WHERE conversation_id IN ($conversation_ids)" >/dev/null || true
  query "DELETE FROM assistant_conversations WHERE id IN ($conversation_ids)" >/dev/null || true
  rm -rf "$temporary_directory"
}
trap cleanup EXIT HUP INT TERM

wait_for_value() {
  description=$1
  sql=$2
  expected=$3
  attempts=10
  actual=''

  while [ "$attempts" -gt 0 ]; do
    actual=$(query "$sql")
    if [ "$actual" = "$expected" ]; then
      printf '%s passed\n' "$description"
      return
    fi
    attempts=$((attempts - 1))
    sleep 1
  done

  printf '%s failed: got %s, expected %s\n' "$description" "${actual:-<missing>}" "$expected" >&2
  return 1
}

seed_callback_contexts() {
  query "
DELETE FROM assistant_conversation_metadata WHERE assistant_conversation_id IN ($conversation_ids);
DELETE FROM assistant_conversation_metrics WHERE assistant_conversation_id IN ($conversation_ids);
DELETE FROM call_contexts WHERE conversation_id IN ($conversation_ids);
DELETE FROM assistant_conversations WHERE id IN ($conversation_ids);
INSERT INTO assistant_conversations (
  id, identifier, assistant_id, assistant_provider_model_id, name,
  project_id, organization_id, source, created_by, status, direction,
  created_actor_type, created_actor_id
) VALUES
  (7100001, 'ci-callback-twilio', 1, 1, 'Twilio callback integration', 1, 1, 'TELEPHONY', 1, 'ACTIVE', 'OUTBOUND', 'project', 1),
  (7100002, 'ci-callback-exotel-busy', 1, 1, 'Exotel busy callback integration', 1, 1, 'TELEPHONY', 1, 'ACTIVE', 'OUTBOUND', 'project', 1),
  (7100003, 'ci-callback-exotel-no-answer', 1, 1, 'Exotel no-answer callback integration', 1, 1, 'TELEPHONY', 1, 'ACTIVE', 'OUTBOUND', 'project', 1),
  (7100004, 'ci-callback-vonage', 1, 1, 'Vonage callback integration', 1, 1, 'TELEPHONY', 1, 'ACTIVE', 'OUTBOUND', 'project', 1),
  (7100005, 'ci-callback-telnyx', 1, 1, 'Telnyx callback integration', 1, 1, 'TELEPHONY', 1, 'ACTIVE', 'OUTBOUND', 'project', 1),
  (7100006, 'ci-callback-asterisk', 1, 1, 'Asterisk callback integration', 1, 1, 'TELEPHONY', 1, 'ACTIVE', 'OUTBOUND', 'project', 1),
  (7100007, 'ci-callback-sip', 1, 1, 'SIP callback integration', 1, 1, 'TELEPHONY', 1, 'ACTIVE', 'OUTBOUND', 'project', 1),
  (7100008, 'ci-callback-vobiz', 1, 1, 'Vobiz callback integration', 1, 1, 'TELEPHONY', 1, 'ACTIVE', 'OUTBOUND', 'project', 1);
INSERT INTO call_contexts (
  id, context_id, assistant_id, conversation_id, project_id, organization_id,
  auth_type, auth_actor_type, auth_actor_id, provider, direction
) VALUES
  (7100001, 'ci-callback-twilio', 1, 7100001, 1, 1, 'project', 'project', 1, 'twilio', 'outbound'),
  (7100002, 'ci-callback-exotel-busy', 1, 7100002, 1, 1, 'project', 'project', 1, 'exotel', 'outbound'),
  (7100003, 'ci-callback-exotel-no-answer', 1, 7100003, 1, 1, 'project', 'project', 1, 'exotel', 'outbound'),
  (7100004, 'ci-callback-vonage', 1, 7100004, 1, 1, 'project', 'project', 1, 'vonage', 'outbound'),
  (7100005, 'ci-callback-telnyx', 1, 7100005, 1, 1, 'project', 'project', 1, 'telnyx', 'outbound'),
  (7100006, 'ci-callback-asterisk', 1, 7100006, 1, 1, 'project', 'project', 1, 'asterisk', 'outbound'),
  (7100007, 'ci-callback-sip', 1, 7100007, 1, 1, 'project', 'project', 1, 'sip', 'outbound'),
  (7100008, 'ci-callback-vobiz', 1, 7100008, 1, 1, 'project', 'project', 1, 'vobiz', 'outbound');
" >/dev/null
}

send_post_callback() {
  provider=$1
  context_id=$2
  content_type=$3
  payload=$4
  response_file="$temporary_directory/$context_id.response"
  http_status=$(curl --silent --show-error --connect-timeout 2 --max-time 10 \
    --output "$response_file" --write-out '%{http_code}' \
    --request POST --header "Content-Type: $content_type" --data-binary "$payload" \
    "$assistant_api_url/v1/talk/$provider/ctx/$context_id/event")

  assert_callback_accepted "$provider" "$http_status" "$response_file"
}

send_get_callback() {
  provider=$1
  context_id=$2
  query_string=$3
  response_file="$temporary_directory/$context_id.response"
  http_status=$(curl --silent --show-error --connect-timeout 2 --max-time 10 \
    --output "$response_file" --write-out '%{http_code}' \
    "$assistant_api_url/v1/talk/$provider/ctx/$context_id/event?$query_string")

  assert_callback_accepted "$provider" "$http_status" "$response_file"
}

assert_callback_accepted() {
  provider=$1
  http_status=$2
  response_file=$3

  if [ "$http_status" != '201' ]; then
    printf '%s callback returned HTTP %s: ' "$provider" "$http_status" >&2
    cat "$response_file" >&2
    return 1
  fi
  printf '%s callback accepted\n' "$provider"
}

send_exotel_callback() {
  context_id=$1
  boundary=$2
  call_sid=$3
  provider_status=$4
  updated_at=$5
  payload_file="$temporary_directory/$context_id.multipart"

  printf '%s\r\n' \
    "--$boundary" \
    'Content-Disposition: form-data; name="CallSid"' \
    '' \
    "$call_sid" \
    "--$boundary" \
    'Content-Disposition: form-data; name="Status"' \
    '' \
    "$provider_status" \
    "--$boundary" \
    'Content-Disposition: form-data; name="DateUpdated"' \
    '' \
    "$updated_at" \
    "--$boundary--" \
    '' > "$payload_file"

  send_post_callback exotel "$context_id" "multipart/form-data; boundary=$boundary" "@$payload_file"
}

assert_failed_callback() {
  provider=$1
  context_id=$2
  conversation_id=$3
  expected_reason=$4
  raw_marker=$5

  wait_for_value "$provider call context status" \
    "SELECT call_status FROM call_contexts WHERE context_id = '$context_id'" \
    'failed' || return 1
  wait_for_value "$provider provider disconnect reason" \
    "SELECT disconnect_reason FROM call_contexts WHERE context_id = '$context_id'" \
    "$expected_reason" || return 1
  wait_for_value "$provider call.status metric" \
    "SELECT value FROM assistant_conversation_metrics WHERE assistant_conversation_id = $conversation_id AND name = 'call.status'" \
    'FAILED' || return 1
  wait_for_value "$provider disconnect type metadata" \
    "SELECT value FROM assistant_conversation_metadata WHERE assistant_conversation_id = $conversation_id AND key = 'disconnect_reason'" \
    'DISCONNECTION_TYPE_ERROR' || return 1
  wait_for_value "$provider raw disconnect metadata" \
    "SELECT CASE WHEN position('$raw_marker' in value) > 0 THEN 'present' ELSE 'missing' END FROM assistant_conversation_metadata WHERE assistant_conversation_id = $conversation_id AND key = 'disconnect_raw_reason'" \
    'present' || return 1
  wait_for_value "$provider conversation status metric" \
    "SELECT value FROM assistant_conversation_metrics WHERE assistant_conversation_id = $conversation_id AND name = 'status'" \
    'FAILED'
}

run_twilio() {
  send_post_callback twilio 'ci-callback-twilio' 'application/x-www-form-urlencoded' \
    'CallSid=CAf64ab88f90f35581dcb16e60f875ea4a&CallStatus=busy' || return 1
  assert_failed_callback twilio 'ci-callback-twilio' 7100001 'busy' 'CallStatus=busy'
}

run_exotel() {
  exotel_failures=0
  send_exotel_callback 'ci-callback-exotel-busy' 'form-data-boundary-mga7whnq347kygeh' \
    'ad5af740ea0d4c9cf816ce81c4e51a93' 'busy' '2026-09-03 20:57:43' || return 1
  send_exotel_callback 'ci-callback-exotel-no-answer' 'form-data-boundary-0qvisod6e1zig9oq' \
    '0cfbf52a7747404d311a085f0ed81a93' 'no-answer' '2026-09-03 17:56:22' || return 1

  assert_failed_callback 'exotel busy' 'ci-callback-exotel-busy' 7100002 'busy' 'busy' || exotel_failures=$((exotel_failures + 1))
  assert_failed_callback 'exotel no-answer' 'ci-callback-exotel-no-answer' 7100003 'no-answer' 'no-answer' || exotel_failures=$((exotel_failures + 1))
  [ "$exotel_failures" -eq 0 ]
}

run_vonage() {
  send_get_callback vonage 'ci-callback-vonage' \
    'status=busy&uuid=vonage-call&duration=0' || return 1
  assert_failed_callback vonage 'ci-callback-vonage' 7100004 'busy' 'status=busy'
}

run_telnyx() {
  send_post_callback telnyx 'ci-callback-telnyx' 'application/json' \
    '{"data":{"event_type":"call.hangup","payload":{"call_control_id":"telnyx-call","hangup_cause":"no_answer","duration":0}}}' || return 1
  assert_failed_callback telnyx 'ci-callback-telnyx' 7100005 'no_answer' 'hangup_cause'
}

run_asterisk() {
  send_post_callback asterisk 'ci-callback-asterisk' 'application/json' \
    '{"type":"ChannelDestroyed","channel":{"id":"asterisk-call"},"cause":17,"cause_txt":"USER_BUSY"}' || return 1
  assert_failed_callback asterisk 'ci-callback-asterisk' 7100006 'USER_BUSY' 'USER_BUSY'
}

run_sip() {
  send_post_callback sip 'ci-callback-sip' 'application/json' \
    '{"event":"failed","call_id":"sip-call","reason":"no_answer"}' || return 1
  assert_failed_callback sip 'ci-callback-sip' 7100007 'no_answer' 'no_answer'
}

run_vobiz() {
  send_post_callback vobiz 'ci-callback-vobiz' 'application/x-www-form-urlencoded' \
    'Event=Hangup&CallUUID=vobiz-call&CallStatus=busy' || return 1
  assert_failed_callback vobiz 'ci-callback-vobiz' 7100008 'busy' 'CallStatus=busy'
}

run_and_record() {
  if ! "$1"; then
    failures=$((failures + 1))
  fi
}

seed_callback_contexts

case "$requested_provider" in
  all)
    run_and_record run_twilio
    run_and_record run_exotel
    run_and_record run_vonage
    run_and_record run_telnyx
    run_and_record run_asterisk
    run_and_record run_sip
    run_and_record run_vobiz
    ;;
  twilio) run_and_record run_twilio ;;
  exotel) run_and_record run_exotel ;;
  vonage) run_and_record run_vonage ;;
  telnyx) run_and_record run_telnyx ;;
  asterisk) run_and_record run_asterisk ;;
  sip) run_and_record run_sip ;;
  vobiz) run_and_record run_vobiz ;;
  *)
    printf 'unsupported telephony callback provider: %s\n' "$requested_provider" >&2
    exit 2
    ;;
esac

if [ "$failures" -ne 0 ]; then
  printf '%s telephony callback provider test(s) failed\n' "$failures" >&2
  exit 1
fi

echo 'Telephony callback integration tests passed'
