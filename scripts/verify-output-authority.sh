#!/usr/bin/env bash
set -euo pipefail

root=${1:?source repository root is required}
candidate=${2:?candidate output directory is required}
root=$(realpath "$root")
candidate=$(realpath -m "$candidate")

test -f "$root/.gooo/capability-effect-checker.gooo"
grep -Eq '^output_scope caller-owned$' "$root/.gooo/capability-effect-checker.gooo"
case "$candidate" in
	"$root"|"$root"/*)
		echo "caller-owned output must be outside source repository" >&2
		exit 1
		;;
esac

if [ -n "${RUNNER_TEMP:-}" ]; then
	runner_temp=$(realpath -m "$RUNNER_TEMP")
	case "$candidate" in
		"$runner_temp"|"$runner_temp"/*) ;;
		*) echo "CI output must be under caller-owned runner temp space" >&2; exit 1 ;;
	esac
fi

test ! -e "$candidate" || test -d "$candidate"
