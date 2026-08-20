#!/usr/bin/env bash
set -u

API_URL="${RAPIDA_CALL_API_URL:-http://localhost:9007/v1/talk/create-phone-call}"
API_KEY="${RAPIDA_API_KEY:-e4da0463dfddcca57c8f4432a64fb6540f3669513f26379d8f1ac9a791e1d11e}"
ASSISTANT_ID="${RAPIDA_ASSISTANT_ID:-2341188802971697152}"
FROM_NUMBER="${RAPIDA_FROM_NUMBER:-5002}"
TO_NUMBER="${RAPIDA_TO_NUMBER:-5001}"
COUNT="${RAPIDA_CALL_COUNT:-1}"
RUN_ID="$(date +%Y%m%d-%H%M%S)"
OUT_DIR="${TMPDIR:-/tmp}/rapida-call-run-${RUN_ID}"

mkdir -p "$OUT_DIR"

echo "Starting ${COUNT} concurrent calls: ${FROM_NUMBER} -> ${TO_NUMBER}"
echo "assistant=${ASSISTANT_ID}"
echo "run_id=${RUN_ID}"
echo "output_dir=${OUT_DIR}"

for i in $(seq 1 "$COUNT"); do
  (
    curl -sS -w "\nHTTP_STATUS=%{http_code}\n" -X POST "$API_URL" \
      -H "Content-Type: application/json" \
      -H "x-api-key: ${API_KEY}" \
      -d "{
        \"assistant\": {
          \"assistantId\": \"${ASSISTANT_ID}\",
          \"version\": \"latest\"
        },
        \"options\": {
          \"speak.voice.id\": \"varsha\"
        },
        \"fromNumber\": \"${FROM_NUMBER}\",
        \"toNumber\": \"${TO_NUMBER}\",
        \"metadata\": {
          \"test\": \"concurrent-sip-call\",
          \"runId\": \"${RUN_ID}\",
          \"callIndex\": ${i},
          \"concurrency\": ${COUNT}
        }
      }" > "${OUT_DIR}/call-${i}.txt"
    echo "finished call ${i}"
  ) &
done

wait

echo
echo "Completed ${COUNT} requests."
echo "Responses:"
for file in "${OUT_DIR}"/call-*.txt; do
  printf "%s " "$(basename "$file" .txt)"
  tail -n 1 "$file"
done
