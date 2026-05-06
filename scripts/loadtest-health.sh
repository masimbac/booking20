#!/usr/bin/env bash
# Example load generator for /v1/health (adjust URL and concurrency).
set -euo pipefail
BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
CONCURRENCY="${CONCURRENCY:-20}"
REQUESTS="${REQUESTS:-200}"

echo "GET ${BASE_URL}/v1/health  (${REQUESTS} reqs, concurrency ${CONCURRENCY})"

seq 1 "${REQUESTS}" | xargs -n1 -P"${CONCURRENCY}" -I{} \
  curl -sf -o /dev/null -w "%{http_code}\n" "${BASE_URL}/v1/health" | sort | uniq -c
