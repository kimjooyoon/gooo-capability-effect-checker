# Gooo capability/effect attenuation v2

## Authority and stages

`.gooo/capability-effect-checker.gooo` is the only semantic authority. It
declares the closed effect vocabulary, capabilities, `METACODE →
GENERATED_CODE → RUNTIME` stage graph, stage grants, direct effect summaries,
case attenuation edges, caller-owned output scope, fixed denominator, cells,
and precedence. Fixture files are observations; they cannot grant themselves
authority.

Go is limited to parsing, typechecking the declared IR, evaluating the
observed call graph/effect sets, and emitting a deterministic checker. All
effect comparison uses the typed `EffectSet`; no permission score or
aggregate score exists.

## Attenuation

For every declared stage edge, the lower-stage grant and attenuation set must
be subsets of the upper-stage grant. The evaluator then checks the observed
generated/runtime effect summary against both the lower grant and the edge.
Generated code cannot create a grant, widen a sibling or ancestor path, or
reverse the stage graph. A known forbidden effect is never downgraded to an
UNKNOWN.

The forbidden set is `REPOSITORY_WRITE`, `REMOTE_MUTATION`, and
`DESTRUCTIVE_DELETE`. `READ_INPUT`, `NETWORK_READ_PINNED`, and
`GENERATE_CALLER_OUTPUT` are also typed effects and must remain within the
declared grant.

## Decision frontier

The only allowed reduction is the explicit `.gooo` fixed point:

`REFUTED > UNKNOWN > CLOSED`

Missing stage grant, missing/ambiguous call or attenuation edge, unknown
generated effect, missing effect summary, missing pinned-network evidence,
and missing caller path scope each produce an UNKNOWN with exactly:
`stage`, `step`, `reason`, `unknown_class`, `next_operation`, and `blocked_by`.

Known forbidden effects, effect/grant amplification, forged grants,
sibling/ancestor scope expansion, and cycles produce REFUTED. A result may
contain both evidence classes; precedence keeps REFUTED above UNKNOWN.

## Corpus contract

The append-only v2 corpus has exactly twelve cases in four-case cohorts:

- FOUNDATION: exact attenuation, zero-effect generated code, caller-owned
  output, pinned network read — all CLOSED.
- COHERENCE: missing grant, missing call edge, unknown generated effect,
  missing path scope — all UNKNOWN.
- REGRESSION: repository-write amplification, remote-mutation amplification,
  destructive ancestor delete, forged grant — all REFUTED.

The expected decision vector is therefore:

`[CLOSED, CLOSED, CLOSED, CLOSED, UNKNOWN, UNKNOWN, UNKNOWN, UNKNOWN,
REFUTED, REFUTED, REFUTED, REFUTED]`.

Proof quotas FOUNDATION/COHERENCE/REGRESSION and indicator quotas
DRIVER/OUTCOME/GUARDRAIL are each 4/4/4. The historical v1 five-case
denominator remains an immutable contract and is not rewritten.

## Evidence authority

PR checks and post-merge `main` checks run on Go 1.27.0. Release lineage is
checked before any release mutation; the next patch is draft-first, then
asset-digest-audited, then published immutable. Runtime authority remains
`repository_writes=0`, `local_test_executions=0`, and
`cross_project_required_gates=0`. Improvement and external utility remain
UNKNOWN until evidence provides a same-scenario source/contract/toolchain
before/after pair or an external utility receipt.
