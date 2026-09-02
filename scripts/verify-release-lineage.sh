#!/usr/bin/env bash
set -euo pipefail

root=${1:?repository root is required}
tag=${RELEASE_TAG:?RELEASE_TAG is required}
commit=$(git -C "$root" rev-parse HEAD)

test "$tag" = "v0.1.2"
tag_object=$(git -C "$root" rev-parse "refs/tags/$tag")
test "$(git -C "$root" cat-file -t "$tag_object")" = tag
tag_commit=$(git -C "$root" rev-list -n 1 "${tag_object}^{commit}")
test "$tag_commit" = "$commit"

if release=$(gh api "repos/${GITHUB_REPOSITORY}/releases/tags/${tag}" 2>/dev/null); then
	echo "release $tag already exists; immutable public releases are append-only" >&2
	exit 1
fi

merged=$(gh api "repos/${GITHUB_REPOSITORY}/commits/${commit}/pulls" --jq '[.[] | select(.merged_at != null)] | length')
test "$merged" -ge 1
test "${GITHUB_REF:-}" = "refs/tags/$tag" || test "${GITHUB_EVENT_NAME:-}" = "workflow_dispatch"
