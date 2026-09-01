#!/usr/bin/env bash
set -euo pipefail

root=${1:?repository root is required}
work=${2:?caller-owned work directory is required}
bin=${3:?checker binary is required}
out="$work/integration-output"
mkdir -p "$out"

before=$(git -C "$root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
"$bin" generate \
	--phase "$root/.gooo/capability-effect-checker.gooo" \
	--corpus-root "$root/examples/corpus" \
	--out "$out" \
	--source-root "$root"
jq -e '.summary == {generated:5,closed:1,unknown:2,refuted:2,failed:0}' "$out/run-report.json" > /dev/null

forbidden="$root/.gooo-repository-write-fixture-output"
if "$bin" generate \
	--phase "$root/.gooo/capability-effect-checker.gooo" \
	--corpus-root "$root/examples/corpus" \
	--out "$forbidden" \
	--source-root "$root"; then
	echo "repository-owned output was accepted" >&2
	exit 1
fi
test ! -e "$forbidden"
after=$(git -C "$root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
test "$before" = "$after"
