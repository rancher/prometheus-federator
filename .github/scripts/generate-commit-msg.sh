#!/bin/bash
set -euo pipefail

# Generates commit message for workflow deployment
# Usage: generate-commit-msg.sh <target-branch> <helm-version> <k3s-versions> <automation-core-ref> <github-sha>
# Output: Prints commit message to stdout

TARGET_BRANCH="$1"
HELM_VERSION="$2"
K3S_VERSIONS="$3"
AUTOMATION_CORE_REF="$4"
GITHUB_SHA="$5"

cat <<EOF
Deploy automation-core workflows

Deployed from automation-core@${GITHUB_SHA:0:7}

Configuration:
- Helm version: ${HELM_VERSION}
- K3S versions: ${K3S_VERSIONS}
- Automation-core ref: @${AUTOMATION_CORE_REF}
- Target branch: ${TARGET_BRANCH}

This PR was automatically created by the workflow deployment system.
EOF
