#!/usr/bin/env bash
# Push Terraform-related GitHub Actions repository Variables (and optional Secrets)
# from the environment or from repo-root `.env-local` using the GitHub CLI.
#
# Prereqs: `gh auth login` (repo scope). Secret sync needs permission to manage Actions secrets.
#
# Usage:
#   ./scripts/sync-github-actions-repo-config.sh
#   ./scripts/sync-github-actions-repo-config.sh -R OWNER/NAME
#   ./scripts/sync-github-actions-repo-config.sh [--ensure-environments]
#   DRY_RUN=1 ./scripts/sync-github-actions-repo-config.sh ...
#
# Uses GH_REPO when -R is passed (see `gh environment`).
set -euo pipefail

ENSURE_ENVIRONMENTS=false

usage() {
  echo "usage: $0 [-R OWNER/REPO] [--ensure-environments]" >&2
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
  -R)
    [[ -n "${2:-}" ]] || usage
    export GH_REPO="$2"
    shift 2
    ;;
  --ensure-environments)
    ENSURE_ENVIRONMENTS=true
    shift
    ;;
  -*)
    usage
    ;;
  *)
    usage
    ;;
  esac
done

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_LOCAL="${ROOT}/.env-local"

require_gh() {
  command -v gh >/dev/null 2>&1 || {
    echo "error: gh not found. Install: https://cli.github.com/ (macOS: make install-gh)" >&2
    exit 1
  }
  gh auth status &>/dev/null || {
    echo "error: not logged in — run: gh auth login (repo scope)." >&2
    exit 1
  }
}

load_backend_env() {
  if [[ -n "${TF_REMOTE_STATE_BUCKET:-}" && -n "${TF_REMOTE_LOCK_TABLE:-}" ]]; then
    BUCKET="${TF_REMOTE_STATE_BUCKET}"
    LOCK="${TF_REMOTE_LOCK_TABLE}"
    REGION="${AWS_REGION:-us-east-1}"
    return
  fi

  if [[ ! -f "$ENV_LOCAL" ]]; then
    echo "error: need TF_REMOTE_STATE_BUCKET + TF_REMOTE_LOCK_TABLE in the environment, or ${ENV_LOCAL} (copy .env-local.example)." >&2
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
}

ensure_environments() {
  local spec
  spec="$(gh repo view --json nameWithOwner -q .nameWithOwner)"
  for env_name in staging production; do
    if [[ -n "${DRY_RUN:-}" ]]; then
      echo "[dry-run] create/update GitHub Environment ${spec} → ${env_name}"
      continue
    fi
    if ! gh api --method PUT "repos/${spec}/environments/${env_name}" --input - <<<'{}' >/dev/null 2>&1; then
      echo "warn: could not PUT environment ${env_name} via API — create \"${env_name}\" under repo Settings → Environments." >&2
    else
      echo "environment ready: ${env_name}"
    fi
  done
  echo "note: add required reviewers under Settings → Environments → production."
}

sync_variables() {
  [[ -n "$BUCKET" ]] || {
    echo "error: missing state bucket (TF_REMOTE_STATE_BUCKET / AWS_S3_BUCKET_NAME)." >&2
    exit 1
  }
  [[ -n "$LOCK" ]] || {
    echo "error: missing lock table (TF_REMOTE_LOCK_TABLE or AWS_DYNAMODB_TABLE_PREFIX)." >&2
    exit 1
  }

  run() {
    if [[ -n "${DRY_RUN:-}" ]]; then
      printf '[dry-run]'
      printf ' %q' "$@"
      echo
    else
      "$@"
    fi
  }

  run gh variable set TERRAFORM_STATE_BUCKET --body "$BUCKET"
  run gh variable set TERRAFORM_LOCK_TABLE --body "$LOCK"
  run gh variable set AWS_REGION --body "$REGION"
  [[ -z "${TF_REMOTE_STATE_BUCKET_STAGING:-}" ]] || run gh variable set TERRAFORM_STATE_BUCKET_STAGING --body "$TF_REMOTE_STATE_BUCKET_STAGING"
  [[ -z "${TF_REMOTE_STATE_BUCKET_PRODUCTION:-}" ]] || run gh variable set TERRAFORM_STATE_BUCKET_PRODUCTION --body "$TF_REMOTE_STATE_BUCKET_PRODUCTION"
  [[ -z "${TF_REMOTE_LOCK_TABLE_STAGING:-}" ]] || run gh variable set TERRAFORM_LOCK_TABLE_STAGING --body "$TF_REMOTE_LOCK_TABLE_STAGING"
  [[ -z "${TF_REMOTE_LOCK_TABLE_PRODUCTION:-}" ]] || run gh variable set TERRAFORM_LOCK_TABLE_PRODUCTION --body "$TF_REMOTE_LOCK_TABLE_PRODUCTION"
  [[ -z "${TF_STATE_KEY_PREFIX:-}" ]] || run gh variable set TERRAFORM_STATE_KEY_PREFIX --body "$TF_STATE_KEY_PREFIX"
}

sync_optional_secrets() {
  _set_secret() {
    local name="$1"
    local val="$2"
    [[ -n "$val" ]] || return 0
    if [[ -n "${DRY_RUN:-}" ]]; then
      echo "[dry-run] gh secret set ${name} -a actions --body <redacted>"
      return 0
    fi
    gh secret set "$name" -a actions --body "$val"
  }

  _set_secret AWS_ROLE_TO_ASSUME_STAGING "${AWS_ROLE_TO_ASSUME_STAGING:-}"
  _set_secret AWS_ROLE_TO_ASSUME_PRODUCTION "${AWS_ROLE_TO_ASSUME_PRODUCTION:-}"
}

require_gh
cd "$ROOT"

if [[ "$ENSURE_ENVIRONMENTS" == true ]]; then
  ensure_environments
fi

load_backend_env
sync_variables
sync_optional_secrets

echo "done: repository Actions variables synced (IAM role secrets only if AWS_ROLE_* were set)."
echo "verify: gh variable list"
