#!/usr/bin/env bash
# Copyright 2026 The cert-manager Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# ping-release-prs.sh
#
# For each PR number given, checks whether a ping referencing the specified
# release has already been posted. If not, posts one. Also follows
# closing-issue links and pings those reporters where not yet done.
#
# Self-reported issues (where the issue reporter is the same as the PR author)
# are skipped automatically.
#
# Idempotent: safe to run multiple times; already-pinged PRs and issues are
# skipped.
#
# Usage:
#   ping-release-prs.sh --release v1.21.0-beta.0 PR1 PR2 ...
#   ping-release-prs.sh --release v1.21.0-beta.0 --repo cert-manager/cert-manager PR1 PR2 ...
#   ping-release-prs.sh --release v1.21.0-beta.0 --dry-run PR1 PR2 ...
#
# Requirements: gh (GitHub CLI), jq

set -euo pipefail

RELEASE=""
REPO="cert-manager/cert-manager"
DRY_RUN=false

usage() {
  cat <<EOF
Usage: $0 --release VERSION [--repo OWNER/REPO] [--dry-run] PR...

  --release VERSION   Release tag to reference (e.g. v1.21.0-beta.0)
  --repo OWNER/REPO   GitHub repository (default: cert-manager/cert-manager)
  --dry-run           Print what would be posted without posting anything
  --help              Show this help
EOF
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --release)  RELEASE="$2";  shift 2 ;;
    --repo)     REPO="$2";     shift 2 ;;
    --dry-run)  DRY_RUN=true;  shift   ;;
    --help|-h)  usage ;;
    *)          break ;;
  esac
done

if [[ -z "$RELEASE" ]]; then
  echo "Error: --release is required" >&2
  usage
fi

if [[ $# -eq 0 ]]; then
  echo "Error: at least one PR number is required" >&2
  usage
fi

RELEASE_URL="https://github.com/${REPO}/releases/tag/${RELEASE}"

# has_ping NUMBER
# Returns 0 (true) if any comment on the issue/PR already contains the release URL.
has_ping() {
  local number="$1"
  local count
  count=$(gh api "repos/${REPO}/issues/${number}/comments?per_page=100" \
    --jq "[.[] | select(.body | contains(\"${RELEASE_URL}\"))] | length")
  [[ "$count" -gt 0 ]]
}

# post_comment NUMBER BODY
post_comment() {
  local number="$1"
  local body="$2"
  if "$DRY_RUN"; then
    echo "    [DRY RUN] would post:"
    echo "$body" | sed 's/^/      /'
  else
    local id
    id=$(gh api "repos/${REPO}/issues/${number}/comments" \
      -f body="$body" --jq '.id')
    echo "    posted comment ${id}"
  fi
}

for pr in "$@"; do
  echo "PR #${pr}"

  # Fetch PR author and linked closing issues in one call.
  pr_data=$(gh pr view "$pr" --repo "$REPO" \
    --json author,closingIssuesReferences \
    --jq '{author: .author.login, is_bot: (.author.is_bot // false), issues: [.closingIssuesReferences[].number]}')

  pr_author=$(echo "$pr_data" | jq -r '.author')
  pr_is_bot=$(echo "$pr_data" | jq -r '.is_bot')

  # Build the @mention for non-bot authors.
  if [[ "$pr_is_bot" == "true" ]] || [[ "$pr_author" == *"[bot]"* ]]; then
    pr_mention=""
  else
    pr_mention="@${pr_author} "
  fi

  # Ping the PR itself.
  if has_ping "$pr"; then
    echo "  already pinged — skipping"
  else
    pr_body="${pr_mention}This change has been included in [${RELEASE}](${RELEASE_URL}), which is now published.

If you are able to install the pre-release and verify that the change works as expected in your environment, that would be much appreciated. Thank you for the contribution."
    echo "  pinging PR"
    post_comment "$pr" "$pr_body"
  fi

  # Ping linked closing issues.
  mapfile -t issues < <(echo "$pr_data" | jq -r '.issues[]')
  for issue in "${issues[@]+"${issues[@]}"}"; do
    echo "  Issue #${issue}"

    issue_reporter=$(gh issue view "$issue" --repo "$REPO" \
      --json author --jq '.author.login')

    # Skip self-reported issues.
    if [[ "$issue_reporter" == "$pr_author" ]]; then
      echo "    self-reported by @${issue_reporter} — skipping"
      continue
    fi

    if has_ping "$issue"; then
      echo "    already pinged — skipping"
    else
      issue_body="@${issue_reporter} The fix or feature you requested has been included in [${RELEASE}](${RELEASE_URL}), which is now published.

If you are able to install the pre-release and verify that it addresses your use case, that would be much appreciated."
      echo "    pinging @${issue_reporter}"
      post_comment "$issue" "$issue_body"
    fi
  done
done

echo "Done."
