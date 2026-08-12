#!/usr/bin/env bash
# Shared setup for push-to-charts scripts. Source this file: source "$(dirname "$0")/common.sh"

# Determine PF_DIR (prometheus-federator root) from this script's location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PF_DIR="${PF_DIR:-$(cd "$SCRIPT_DIR/../../.." && pwd)}"

# Required: path to a local rancher/charts clone
CHARTS_DIR="${CHARTS_DIR:-}"

# Remote name for rancher/charts in CHARTS_DIR (may differ locally if using a fork)
CHARTS_REMOTE="${CHARTS_REMOTE:-origin}"

# Skip git commits, push, and PR creation when true
DRY_RUN="${DRY_RUN:-false}"

# Path (relative to packages/) and chart name for prometheus-federator in rancher/charts.
PACKAGE="rancher-monitoring/prometheus-federator"
CHART_NAME="prometheus-federator"

# Map of prometheus-federator major version → rancher/charts target branch.
# Add a new entry here when a new prometheus-federator/Rancher version pair is created.
declare -A CHARTS_BRANCH_MAP=(
  ["7"]="dev-v2.15"
  ["6"]="dev-v2.14"
  ["5"]="dev-v2.13"
  ["4"]="dev-v2.12"
  ["3"]="dev-v2.11"
  ["2"]="dev-v2.10"
  ["1"]="dev-v2.9"
)

# Resolve the rancher/charts target branch for a given prometheus-federator tag (e.g. v7.0.1-rc.5).
# Prints the branch name and exits 1 if the major version is not in the map.
get_charts_branch() {
  local tag="${1#v}"  # strip leading v
  local major
  major=$(echo "$tag" | cut -d. -f1)
  local branch="${CHARTS_BRANCH_MAP[$major]:-}"
  if [ -z "$branch" ]; then
    echo "ERROR: No rancher/charts branch configured for prometheus-federator major version '$major' (tag: $1)" >&2
    echo "ERROR: Add an entry to CHARTS_BRANCH_MAP in common.sh" >&2
    exit 1
  fi
  echo "$branch"
}

# Write to GitHub step summary if available, and always print to stdout
summary() {
  if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
    echo "$@" >> "$GITHUB_STEP_SUMMARY"
  fi
  echo "$@"
}

require_var() {
  local var="$1"
  if [ -z "${!var:-}" ]; then
    echo "ERROR: $var is required" >&2
    exit 1
  fi
}

require_charts_dir() {
  require_var CHARTS_DIR
  if [ ! -d "$CHARTS_DIR" ]; then
    echo "ERROR: CHARTS_DIR '$CHARTS_DIR' does not exist" >&2
    exit 1
  fi
}

# Returns 0 if the given TARGET_BRANCH is frozen, 1 otherwise.
is_branch_frozen() {
  local target_branch="$1"
  local freeze_manifest="${2:-/tmp/pf-push/code-freeze.yaml}"
  if [ ! -f "$freeze_manifest" ]; then
    return 1
  fi
  local manifest_branch_name
  manifest_branch_name=$(echo "$target_branch" | sed 's|dev-v|release/v|')
  local manifest_ref="refs/heads/$manifest_branch_name"
  local result
  result=$(yq e ".spec.forProvider.conditions[].refName[].include[] | select(. == \"$manifest_ref\")" "$freeze_manifest")
  [ -n "$result" ]
}

# Commit all changes in CHARTS_DIR if any exist. Does nothing if tree is clean.
commit_if_changed() {
  local message="$1"
  if git -C "$CHARTS_DIR" diff --quiet --exit-code && [ -z "$(git -C "$CHARTS_DIR" status --porcelain)" ]; then
    return 0
  fi
  git -C "$CHARTS_DIR" add .
  git -C "$CHARTS_DIR" commit -m "$message"
}
