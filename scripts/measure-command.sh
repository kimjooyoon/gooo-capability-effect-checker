#!/usr/bin/env bash
set -euo pipefail

output=${1:?timing output is required}
shift
if [ "${1:-}" != "--" ]; then
	echo "usage: measure-command.sh output -- command [args...]" >&2
	exit 2
fi
shift
mkdir -p "$(dirname "$output")"
raw="$output.raw"
start=$(date +%s%3N)
set +e
/usr/bin/time -v -o "$raw" "$@"
status=$?
set -e
end=$(date +%s%3N)
wall=$((end - start))
if [ "$wall" -lt 1 ]; then wall=1; fi
rss_kib=$(awk -F: '/Maximum resident set size/ {gsub(/^[[:space:]]+/, "", $2); print $2; exit}' "$raw")
rss_kib=${rss_kib:-0}
jq -n --argjson wall_ms "$wall" --argjson exit_code "$status" --argjson peak_rss_kib "$rss_kib" \
	'{wall_ms:$wall_ms,exit_code:$exit_code,peak_rss_kib:$peak_rss_kib,peak_rss_bytes:($peak_rss_kib * 1024)}' > "$output"
exit "$status"
