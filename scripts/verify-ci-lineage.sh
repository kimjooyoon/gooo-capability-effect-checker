#!/usr/bin/env bash
set -euo pipefail

root=${1:?repository root is required}
subject_sha=${2:?checked out subject SHA is required}
event=${GITHUB_EVENT_NAME:?GITHUB_EVENT_NAME is required in CI}
test -d "$root/.git"
test "$(git -C "$root" rev-parse HEAD)" = "$subject_sha"

case "$event" in
	pull_request)
		test -n "${GITHUB_EVENT_PULL_REQUEST_HEAD_SHA:-$subject_sha}"
		;;
	push)
		test "${GITHUB_REF:-}" = "refs/heads/main"
		merged=$(gh api "repos/${GITHUB_REPOSITORY}/commits/${subject_sha}/pulls" --jq '[.[] | select(.merged_at != null)] | length')
		test "$merged" -ge 1
		;;
	workflow_dispatch)
		test "${GITHUB_REF:-}" = "refs/heads/main"
		merged=$(gh api "repos/${GITHUB_REPOSITORY}/commits/${subject_sha}/pulls" --jq '[.[] | select(.merged_at != null)] | length')
		test "$merged" -ge 1
		;;
	*)
		echo "unsupported CI event for PR-first authority: $event" >&2
		exit 1
		;;
esac
