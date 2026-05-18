#!/usr/bin/env bash
# Generates infra/terraform/.backend/{staging,production}.hcl.
#
# Local: sources repo-root `.env-local` (see `.env-local.example`).
# CI: set TF_REMOTE_STATE_BUCKET + TF_REMOTE_LOCK_TABLE (+ AWS_REGION) in the environment
# (e.g. from GitHub repository variables), then run this script.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_LOCAL="${ROOT}/.env-local"
OUT="${ROOT}/infra/terraform/.backend"
mkdir -p "$OUT"

if [[ -n "${TF_REMOTE_STATE_BUCKET:-}" && -n "${TF_REMOTE_LOCK_TABLE:-}" ]]; then
  BUCKET="${TF_REMOTE_STATE_BUCKET}"
  LOCK="${TF_REMOTE_LOCK_TABLE}"
  REGION="${AWS_REGION:-us-east-1}"
else
  if [[ ! -f "$ENV_LOCAL" ]]; then
    echo "error: either set TF_REMOTE_STATE_BUCKET + TF_REMOTE_LOCK_TABLE, or provide ${ENV_LOCAL} (copy .env-local.example)." >&2
    exit 1
  fi

  set -a
  # shellcheck disable=SC1090
  source "$ENV_LOCAL"
  set +a

  BUCKET="${TF_REMOTE_STATE_BUCKET:-${AWS_S3_BUCKET_NAME:-}}"
  LOCK="${TF_REMOTE_LOCK_TABLE:-}"

  if [[ -z "$LOCK" && -n "${AWS_DYNAMODB_TABLE_PREFIX:-}" ]]; then
    LOCK="${AWS_DYNAMODB_TABLE_PREFIX}-terraform-locks"
  fi

  REGION="${AWS_REGION:-us-east-1}"
fi

if [[ -z "$BUCKET" ]]; then
  echo "error: TF_REMOTE_STATE_BUCKET / AWS_S3_BUCKET_NAME (.env-local) or TF_REMOTE_STATE_BUCKET (env)." >&2
  exit 1
fi
if [[ -z "$LOCK" ]]; then
  echo "error: TF_REMOTE_LOCK_TABLE (.env-local) or TF_REMOTE_LOCK_TABLE (env), or derive via AWS_DYNAMODB_TABLE_PREFIX in .env-local." >&2
  exit 1
fi

write_hcl() {
  local name="$1"
  local object_key="$2"
  cat >"${OUT}/${name}.hcl" <<EOF
bucket         = "${BUCKET}"
key            = "${object_key}"
region         = "${REGION}"
encrypt        = true
dynamodb_table = "${LOCK}"
EOF
}

write_hcl staging "booking/staging/terraform.tfstate"
write_hcl production "booking/production/terraform.tfstate"

echo "wrote ${OUT}/staging.hcl and production.hcl (bucket=${BUCKET})"
