#!/usr/bin/env bash
set -euo pipefail

root=${1:?repository root is required}
work=${2:?caller-owned work directory is required}
bin=${3:?checker binary is required}
mkdir -p "$work"
output="$work/conformance-output"
replay="$work/conformance-replay"
mkdir -p "$output" "$replay"

before=$(git -C "$root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
start=$(date +%s%3N)
/usr/bin/time -v -o "$work/generator-time.raw" "$bin" generate \
	--phase "$root/.gooo/capability-effect-checker.gooo" \
	--corpus-root "$root/examples/corpus" \
	--out "$output" \
	--source-root "$root"
"$bin" generate \
	--phase "$root/.gooo/capability-effect-checker.gooo" \
	--corpus-root "$root/examples/corpus" \
	--out "$replay" \
	--source-root "$root"
end=$(date +%s%3N)
wall=$((end - start))
if [ "$wall" -lt 1 ]; then wall=1; fi

for relative in semantic-ir.json generated/checker.go run-report.json human-report.md; do
	cmp -s "$output/$relative" "$replay/$relative"
done

jq -e '
	.schema == "gooo/capability-effect-checker/run-report/v1" and
	.summary == {generated:5,closed:1,unknown:2,refuted:2,failed:0} and
	(.cases | length == 5) and
	.authority == {repository_writes:0,local_test_executions:0,cross_project_required_gates:0} and
	(.generated_artifacts | length == 1) and
	(.generated_artifacts[0].path == "generated/checker.go") and
	([.cases[] | select(.decision == "UNKNOWN") | .unknowns[] |
		(.stage != "" and .step != "" and .reason != "" and .unknown_class != "" and .next_operation != "" and (.blocked_by | length > 0))] | all) and
	([.cases[] | select(.id == "safe-generator") | .decision == "CLOSED" and .inferred_effects == ["READ_INPUT","WRITE_CALLER_OUTPUT"] and (.offending_call_paths | length == 0)] | all) and
	([.cases[] | select(.id == "repository-write-escalation") | .decision == "REFUTED" and .inferred_effects == ["READ_INPUT","REPOSITORY_WRITE"] and any(.offending_call_paths[]; .effect == "REPOSITORY_WRITE" and .path == ["generator"])] | all) and
	([.cases[] | select(.id == "indirect-missing-grant") | .decision == "UNKNOWN" and .unknowns[0].unknown_class == "INDIRECT_GRANT" and any(.offending_call_paths[]; .issue == "missing_indirect_grant" and .path == ["generator","helper"])] | all) and
	([.cases[] | select(.id == "indirect-repository-write") | .decision == "REFUTED" and .inferred_effects == ["READ_INPUT","REPOSITORY_WRITE","WRITE_CALLER_OUTPUT"] and any(.offending_call_paths[]; .effect == "REPOSITORY_WRITE" and .path == ["generator","helper"])] | all) and
	([.cases[] | select(.id == "external-oracle") | .decision == "UNKNOWN" and .unknowns[0].unknown_class == "EXTERNAL_ORACLE" and .external_dependencies == ["generator:github-ruleset"]] | all)
' "$output/run-report.json" > /dev/null

after=$(git -C "$root" status --porcelain=v1 -z --untracked-files=all | sha256sum | awk '{print $1}')
test "$before" = "$after"
output_files=$(find "$output" -type f | wc -l | tr -d ' ')
output_bytes=$(find "$output" -type f -print0 | xargs -0 stat -c '%s' | awk '{total += $1} END {print total + 0}')
generated_files=$(find "$output/generated" -type f | wc -l | tr -d ' ')
generated_bytes=$(find "$output/generated" -type f -print0 | xargs -0 stat -c '%s' | awk '{total += $1} END {print total + 0}')
rss_kib=$(awk -F: '/Maximum resident set size/ {gsub(/^[[:space:]]+/, "", $2); print $2; exit}' "$work/generator-time.raw")
rss_kib=${rss_kib:-0}
rss_bytes=$((rss_kib * 1024))
jq -n \
	--argjson wall "$wall" --argjson peak_rss_kib "$rss_kib" --argjson peak_rss_bytes "$rss_bytes" \
	--argjson output_files "$output_files" --argjson output_bytes "$output_bytes" \
	--argjson generated_files "$generated_files" --argjson generated_bytes "$generated_bytes" \
	'{schema:"gooo/capability-effect-checker/conformance/v1",tests:{total:5,selected:5,executed:5,reused:0,failed:0,unknown:2},cases:{closed:1,unknown:2,refuted:2},outputs:{count:$output_files,bytes:$output_bytes,generated_artifacts:{count:$generated_files,bytes:$generated_bytes}},resources:{peak_rss_kib:$peak_rss_kib,peak_rss_bytes:$peak_rss_bytes},wall_ms:$wall}' \
	> "$work/conformance-report.json"
