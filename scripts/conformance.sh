#!/usr/bin/env bash
set -euo pipefail

root=${1:?repository root is required}
work=${2:?caller-owned work directory is required}
bin=${3:?checker binary is required}
mkdir -p "$work"
output="$work/conformance-output"
replay="$work/conformance-replay"
bash "$(dirname "$0")/verify-output-authority.sh" "$root" "$output"
bash "$(dirname "$0")/verify-output-authority.sh" "$root" "$replay"
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
	.schema == "gooo/capability-effect-checker/run-report/v2" and
	.summary == {generated:12,closed:4,unknown:4,refuted:4,failed:0} and
	.decision_vector == ["CLOSED","CLOSED","CLOSED","CLOSED","UNKNOWN","UNKNOWN","UNKNOWN","UNKNOWN","REFUTED","REFUTED","REFUTED","REFUTED"] and
	.effect_vectors == [
		{id:"exact-attenuation",effects:["GENERATE_CALLER_OUTPUT","READ_INPUT"]},
		{id:"zero-effect-generated",effects:["GENERATE_CALLER_OUTPUT","READ_INPUT"]},
		{id:"caller-owned-output",effects:["GENERATE_CALLER_OUTPUT","READ_INPUT"]},
		{id:"pinned-network-read",effects:["NETWORK_READ_PINNED","READ_INPUT"]},
		{id:"missing-grant",effects:["GENERATE_CALLER_OUTPUT","NETWORK_READ_PINNED","READ_INPUT"]},
		{id:"missing-call-edge",effects:["GENERATE_CALLER_OUTPUT","READ_INPUT"]},
		{id:"unknown-generated-effect",effects:["GENERATED_EFFECT_UNKNOWN","GENERATE_CALLER_OUTPUT","READ_INPUT"]},
		{id:"missing-path-scope",effects:["GENERATE_CALLER_OUTPUT","READ_INPUT"]},
		{id:"repo-write-amplification",effects:["GENERATE_CALLER_OUTPUT","READ_INPUT","REPOSITORY_WRITE"]},
		{id:"remote-mutation-amplification",effects:["GENERATE_CALLER_OUTPUT","READ_INPUT","REMOTE_MUTATION"]},
		{id:"destructive-ancestor-delete",effects:["DESTRUCTIVE_DELETE","GENERATE_CALLER_OUTPUT","READ_INPUT"]},
		{id:"forged-grant",effects:["GENERATE_CALLER_OUTPUT","READ_INPUT"]}
	] and
	(.cases | length == 12) and
	(.cells | length == 12) and
	.proof_counts == {FOUNDATION:4,COHERENCE:4,REGRESSION:4} and
	.indicator_counts == {DRIVER:4,OUTCOME:4,GUARDRAIL:4} and
	.cohort_counts == {FOUNDATION:4,COHERENCE:4,REGRESSION:4} and
	.fixed_point == "explicit" and
	.precedence == ["REFUTED","UNKNOWN","CLOSED"] and
	.authority == {repository_writes:0,local_test_executions:0,cross_project_required_gates:0} and
	.evidence.improvement.status == "UNKNOWN" and
	.evidence.external_utility.status == "UNKNOWN" and
	(.generated_artifacts | length == 1) and
	(.generated_artifacts[0].path == "generated/checker.go") and
	([.cases[] | select(.decision == "UNKNOWN") | .unknowns[] |
		(.stage != "" and .step != "" and .reason != "" and .unknown_class != "" and .next_operation != "" and (.blocked_by | length > 0))] | all) and
	([.cases[] | select(.id == "exact-attenuation") | .decision == "CLOSED" and .inferred_effects == ["GENERATE_CALLER_OUTPUT","READ_INPUT"] and (.offending_call_paths | length == 0)] | all) and
	([.cases[] | select(.id == "zero-effect-generated") | .stage_results[] | select(.stage == "GENERATED_CODE") | .direct_effects == []] | all) and
	([.cases[] | select(.id == "caller-owned-output") | .decision == "CLOSED" and .output_scope == "caller-owned"] | all) and
	([.cases[] | select(.id == "pinned-network-read") | .decision == "CLOSED" and .inferred_effects == ["NETWORK_READ_PINNED","READ_INPUT"]] | all) and
	([.cases[] | select(.id == "missing-grant") | .decision == "UNKNOWN" and any(.unknowns[]; .unknown_class == "MISSING_GRANT")] | all) and
	([.cases[] | select(.id == "missing-call-edge") | .decision == "UNKNOWN" and any(.unknowns[]; .unknown_class == "MISSING_CALL_EDGE")] | all) and
	([.cases[] | select(.id == "unknown-generated-effect") | .decision == "UNKNOWN" and any(.unknowns[]; .unknown_class == "UNKNOWN_GENERATED_EFFECT")] | all) and
	([.cases[] | select(.id == "missing-path-scope") | .decision == "UNKNOWN" and any(.unknowns[]; .unknown_class == "MISSING_PATH_SCOPE")] | all) and
	([.cases[] | select(.id == "repo-write-amplification") | .decision == "REFUTED" and any(.offending_call_paths[]; .effect == "REPOSITORY_WRITE" and .path == ["metacode:generator","generated:generator"])] | all) and
	([.cases[] | select(.id == "remote-mutation-amplification") | .decision == "REFUTED" and any(.offending_call_paths[]; .effect == "REMOTE_MUTATION" and .path == ["metacode:generator","generated:generator","runtime:generator"])] | all) and
	([.cases[] | select(.id == "destructive-ancestor-delete") | .decision == "REFUTED" and any(.offending_call_paths[]; .issue == "destructive_ancestor_delete" and .path == ["metacode:generator","generated:generator","metacode:cleanup"])] | all) and
	([.cases[] | select(.id == "forged-grant") | .decision == "REFUTED" and any(.offending_call_paths[]; .issue == "forged_grant" and .path == ["metacode:generator","generated:generator"])] | all)
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
	'{schema:"gooo/capability-effect-checker/conformance/v2",tests:{total:12,selected:12,executed:12,reused:0,failed:0,unknown:4},cases:{closed:4,unknown:4,refuted:4},outputs:{count:$output_files,bytes:$output_bytes,generated_artifacts:{count:$generated_files,bytes:$generated_bytes}},resources:{peak_rss_kib:$peak_rss_kib,peak_rss_bytes:$peak_rss_bytes},wall_ms:$wall}' \
	> "$work/conformance-report.json"
