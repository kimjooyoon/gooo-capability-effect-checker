# Gooo capability effect checker v1

## Scope

The `.gooo` source is the only semantic authority. It declares the finite
effect vocabulary, capability names, per-case grants, risk effects, decision
precedence, six-field UNKNOWN frontier, fixed denominator, and generation
steps. Go implements only the line parser, the call-graph executor, and the
generated Go emitter.

The checker is deliberately an exact-set analysis. For a root function it
walks the reachable call graph, unions each reachable function's direct
effects, and emits a lexicographically deterministic set. It compares that
set with the root grant and also checks every reachable function's grant.
There is no permission score, percentage, inferred allowance, or aggregate
credit.

## Decisions

The precedence declared in `.gooo` is `REFUTED > UNKNOWN > CLOSED`.
`REFUTED` is a known contradiction: a declared grant omits an observed effect,
an effect is outside the declared vocabulary, a known mutation escapes a
grant, or the call graph is cyclic. `UNKNOWN` preserves a missing indirect
grant when the callee is known but no grant was declared, an unresolved call
target, or an unavailable external oracle. A missing indirect grant that
reaches `REPOSITORY_WRITE`, `CI_MUTATION`, or `RELEASE_MUTATION` is instead
`REFUTED`, because the unsafe effect is known.

Every UNKNOWN has exactly these fields: `stage`, `step`, `reason`,
`unknown_class`, `next_operation`, and `blocked_by`. The reducer never hides a
REFUTED issue behind an UNKNOWN one.

## Call paths

The executor records the shortest root-to-function path for each issue, with
lexicographic tie-breaking over sorted call edges. A path is evidence for the
exact effect or missing grant; it is not a score or a risk ranking.

## Corpus and boundary

The fixed denominator is five cases: one safe generator (`CLOSED`), two known
contradictions (`REFUTED`), and two unknown frontiers (`UNKNOWN`). The corpus
contains both indirect-grant outcomes and an external-oracle dependency.
CI runs the real generator into a caller-owned temporary directory, repeats it
for deterministic replay, compiles the emitted checker, and attempts a
repository-owned output path that must be rejected before any write. The input
repository remains unchanged.

## Evidence

GitHub Actions with Go 1.27 is the validation authority. CI records exact
integer counts and durations for Go/Gooo files and physical lines,
subdirectories, regular files, generated artifacts and bytes, peak RSS,
compile/build/test/conformance/integration wall time, and total/selected/
executed/reused/failed/unknown corpus tests. No local execution is used as a
success claim. Improvement is `UNKNOWN` until the same scenario, source,
contract, and toolchain digests have a before/after integer pair.
