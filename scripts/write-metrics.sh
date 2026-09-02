#!/usr/bin/env bash
set -euo pipefail

root=${1:?repository root is required}
work=${2:?CI work directory is required}
output=${3:?metrics output is required}

go_files=$(git -C "$root" ls-files '*.go' | wc -l | tr -d ' ')
gooo_files=$(git -C "$root" ls-files '*.gooo' | wc -l | tr -d ' ')
go_lines=$(git -C "$root" ls-files '*.go' -z | xargs -0 awk '{count += 1} END {print count + 0}')
gooo_lines=$(git -C "$root" ls-files '*.gooo' -z | xargs -0 awk '{count += 1} END {print count + 0}')
regular_files=$(git -C "$root" ls-files | awk '$0 != "README.md" {count += 1} END {print count + 0}')
subdirectories=$(git -C "$root" ls-files | awk -F/ 'NF > 1 {for (i=1; i<NF; i++) {path=""; for (j=1; j<=i; j++) path=path (j==1 ? "" : "/") $j; seen[path]=1}} END {count=0; for (path in seen) count++; print count + 0}')

timing_peak=$(jq -s 'map(.peak_rss_bytes) | max' "$work"/timing/*.json)
timing_peak=${timing_peak:-0}
peak_kib=$((timing_peak / 1024))
phase_digest="sha256:$(sha256sum "$root/.gooo/capability-effect-checker.gooo" | awk '{print $1}')"
semantic_ir_digest=$(jq -r '.semantic_ir_digest' "$work/conformance-output/run-report.json")
run_report_digest="sha256:$(sha256sum "$work/conformance-output/run-report.json" | awk '{print $1}')"
source_commit=$(git -C "$root" rev-parse HEAD)

jq -n \
	--arg source_commit "$source_commit" \
	--arg phase_digest "$phase_digest" \
	--arg semantic_ir_digest "$semantic_ir_digest" \
	--arg run_report_digest "$run_report_digest" \
	--argjson go_files "$go_files" --argjson gooo_files "$gooo_files" \
	--argjson go_lines "$go_lines" --argjson gooo_lines "$gooo_lines" \
	--argjson regular_files "$regular_files" --argjson subdirectories "$subdirectories" \
	--argjson output_count "$(jq -r '.outputs.count' "$work/conformance-report.json")" \
	--argjson output_bytes "$(jq -r '.outputs.bytes' "$work/conformance-report.json")" \
	--argjson generated_count "$(jq -r '.outputs.generated_artifacts.count' "$work/conformance-report.json")" \
	--argjson generated_bytes "$(jq -r '.outputs.generated_artifacts.bytes' "$work/conformance-report.json")" \
	--argjson peak_bytes "$timing_peak" --argjson peak_kib "$peak_kib" \
	--argjson compile_ms "$(jq -r '.wall_ms' "$work/timing/compile.json")" \
	--argjson build_ms "$(jq -r '.wall_ms' "$work/timing/build.json")" \
	--argjson test_ms "$(jq -r '.wall_ms' "$work/timing/test.json")" \
	--argjson conformance_ms "$(jq -r '.wall_ms' "$work/conformance-report.json")" \
	--argjson integration_ms "$(jq -r '.wall_ms' "$work/timing/integration.json")" \
	--argjson total "$(jq -r '.tests.total' "$work/conformance-report.json")" \
	--argjson selected "$(jq -r '.tests.selected' "$work/conformance-report.json")" \
	--argjson executed "$(jq -r '.tests.executed' "$work/conformance-report.json")" \
	--argjson reused "$(jq -r '.tests.reused' "$work/conformance-report.json")" \
	--argjson failed "$(jq -r '.tests.failed' "$work/conformance-report.json")" \
	--argjson unknown "$(jq -r '.tests.unknown' "$work/conformance-report.json")" \
	'{schema:"gooo/capability-effect-checker/ci-metrics/v2",inventory:{go_files:$go_files,gooo_files:$gooo_files,go_physical_lines:$go_lines,gooo_physical_lines:$gooo_lines,subdirectories:$subdirectories,regular_files:$regular_files,root_readme_inventory_excluded:1},outputs:{count:$output_count,bytes:$output_bytes,generated_artifacts:{count:$generated_count,bytes:$generated_bytes}},resources:{peak_rss_bytes:$peak_bytes,peak_rss_kib:$peak_kib},wall_ms:{compile:$compile_ms,build:$build_ms,test:$test_ms,conformance:$conformance_ms,integration:$integration_ms},tests:{total:$total,selected:$selected,executed:$executed,reused:$reused,failed:$failed,unknown:$unknown},authority:{repository_writes:0,local_test_executions:0,cross_project_required_gates:0},lineage:{source_commit:$source_commit,phase_digest:$phase_digest,semantic_ir_digest:$semantic_ir_digest,run_report_digest:$run_report_digest},improvement:{status:"UNKNOWN",reason:"NO_SAME_SCENARIO_SOURCE_CONTRACT_TOOLCHAIN_BEFORE_AFTER_PAIR"},external_utility:{status:"UNKNOWN",reason:"NO_EXTERNAL_UTILITY_EVIDENCE"}}' \
	> "$output"
