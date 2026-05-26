#!/usr/bin/env bash
# Generates infra/terraform/.backend/{staging,production}.hcl.
#
# Local: repo-root `.env-local` (see `.env-local.example`).
# CI (or shell): export TF_REMOTE_STATE_BUCKET + TF_REMOTE_LOCK_TABLE (+ AWS_REGION),
#   or use per-environment buckets TF_REMOTE_STATE_BUCKET_STAGING / TF_REMOTE_STATE_BUCKET_PRODUCTION
#   with one shared TF_REMOTE_LOCK_TABLE or split TF_REMOTE_LOCK_TABLE_STAGING / TF_REMOTE_LOCK_TABLE_PRODUCTION.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_LOCAL="${ROOT}/.env-local"
OUT="${ROOT}/infra/terraform/.backend"
mkdir -p "$OUT"

if [[ -f "$ENV_LOCAL" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ENV_LOCAL"
  set +a
fi

REGION="${AWS_REGION:-us-east-1}"

# Default bucket / lock (one shared bucket: different state keys per env)
BUCKET_DEFAULT="${TF_REMOTE_STATE_BUCKET:-${AWS_S3_BUCKET_NAME:-}}"
LOCK_DEFAULT="${TF_REMOTE_LOCK_TABLE:-}"

if [[ -z "$LOCK_DEFAULT" && -n "${AWS_DYNAMODB_TABLE_PREFIX:-}" ]]; then
  LOCK_DEFAULT="${AWS_DYNAMODB_TABLE_PREFIX}-terraform-locks"
fi

BUCKET_STAGING="${TF_REMOTE_STATE_BUCKET_STAGING:-$BUCKET_DEFAULT}"
BUCKET_PRODUCTION="${TF_REMOTE_STATE_BUCKET_PRODUCTION:-$BUCKET_DEFAULT}"

LOCK_STAGING="${TF_REMOTE_LOCK_TABLE_STAGING:-$LOCK_DEFAULT}"
LOCK_PRODUCTION="${TF_REMOTE_LOCK_TABLE_PRODUCTION:-$LOCK_DEFAULT}"

if [[ -z "$BUCKET_STAGING" ]] || [[ -z "$BUCKET_PRODUCTION" ]]; then
  echo "error: set TF_REMOTE_STATE_BUCKET (or TF_REMOTE_*_STAGING/PRODUCTION) in env or ${ENV_LOCAL}." >&2
  exit 1
fi
if [[ -z "$LOCK_STAGING" ]] || [[ -z "$LOCK_PRODUCTION" ]]; then
  echo "error: set TF_REMOTE_LOCK_TABLE (or per-env *_STAGING/_PRODUCTION) in env or ${ENV_LOCAL}." >&2
  exit 1
fi

STATE_KEY_PREFIX="${TF_STATE_KEY_PREFIX:-booking}"

write_hcl() {
  local name="$1"
  local bucket="$2"
  local object_key="$3"
  local lock_table="$4"
  cat >"${OUT}/${name}.hcl" <<EOF
bucket         = "${bucket}"
key            = "${object_key}"
region         = "${REGION}"
encrypt        = true
dynamodb_table = "${lock_table}"
EOF
}

write_hcl staging "${BUCKET_STAGING}" "${STATE_KEY_PREFIX}/staging/terraform.tfstate" "${LOCK_STAGING}"
write_hcl production "${BUCKET_PRODUCTION}" "${STATE_KEY_PREFIX}/production/terraform.tfstate" "${LOCK_PRODUCTION}"

echo "wrote ${OUT}/staging.hcl (bucket=${BUCKET_STAGING}) and production.hcl (bucket=${BUCKET_PRODUCTION})"
