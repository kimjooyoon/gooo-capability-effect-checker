# Gooo capability/effect attenuation checker

This repository extends the existing checker with one authority-preserving
pipeline:

`metacode (.gooo) → generated code → runtime`

The actual `.gooo` phase is the sole authority for capability grants, stage
bindings, direct effect summaries, attenuation edges, caller-owned output
scope, the denominator, proof/indicator cells, and decision precedence. Go
provides only the parser, typechecker, evaluator, and deterministic emitter.

The typed effect vocabulary is exactly:

`READ_INPUT`, `NETWORK_READ_PINNED`, `GENERATE_CALLER_OUTPUT`,
`REPOSITORY_WRITE`, `REMOTE_MUTATION`, `DESTRUCTIVE_DELETE`.

Every lower stage is checked against its upper grant and its declared
attenuation edge. Generated or runtime code cannot mint a grant, widen a
sibling/ancestor path, write the repository, mutate remotely, or delete
destructively. Known forbidden effects are `REFUTED`; missing grants, missing
call/attenuation edges, unknown generated effects, and missing path scope are
`UNKNOWN` with exactly six fields. Decisions reduce as
`REFUTED > UNKNOWN > CLOSED` at an explicit fixed point.

The v2 denominator is exactly 12 cells: four FOUNDATION, four COHERENCE, and
four REGRESSION cases. Proof quotas are FOUNDATION/COHERENCE/REGRESSION 4/4/4;
indicator quotas are DRIVER/OUTCOME/GUARDRAIL 4/4/4. No aggregate score is
computed.

The previous five-case denominator is preserved in
[`contracts/denominator-v1.json`](contracts/denominator-v1.json), and v2 is an
append-only contract in
[`contracts/denominator-v2.json`](contracts/denominator-v2.json). The public
immutable v0.1.1 artifact is not rewritten. Its audit is recorded in
[`docs/release-history-v1.md`](docs/release-history-v1.md).
The failed v0.1.2 draft is intentionally preserved and burned; its exact
asset-audit receipt is in
[`contracts/release-receipt-v0.1.2.json`](contracts/release-receipt-v0.1.2.json).

## CI authority

Go 1.27.0, PR checks, and the post-merge `main` check are the only validation
authority. Local verification and generation are intentionally not evidence
(`0`). CI records exact integer wall time, peak RSS, inventory, output and
generated-artifact counts/bytes, exact decision/effect vectors, and the three
zero authority values: repository writes, local test executions, and
cross-project required gates.

The release workflow checks PR-first lineage and caller-owned output before
any remote mutation, creates the next patch release as a draft, audits its
asset digests, and only then publishes the immutable release. After the burned
v0.1.2 draft, the fresh next patch is v0.1.3. Existing public releases are
append-only.

## CI command

```text
go run ./cmd/gooo-capability-effect-checker generate \
  --phase .gooo/capability-effect-checker.gooo \
  --corpus-root examples/corpus \
  --out /caller-owned/output \
  --source-root .
```

The generated report contains stage results, attenuation evidence, exact
vectors, shortest offending paths, and complete UNKNOWN frontiers. The
command is invoked by CI into caller-owned temporary space; the command above
is documentation only and is not local evidence.
