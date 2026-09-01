# Gooo capability effect checker

This repository defines an executable Gooo capability/effect checker. Its
`.gooo` metacode owns the effect vocabulary, capability grants, denominator,
decision precedence, UNKNOWN frontier, and generation plan. Go is only the
parser, executor, and emitter for that declaration.

The first `main` commit is the `BOOTSTRAP_EXCEPTION` from
`gooo-repository-bootstrap` v0.1.1. Substantive implementation is introduced
through one pull request and is validated by GitHub Actions with Go 1.27.

The checker computes the exact transitive effect set reachable from a root
function, compares it with the declared grants, and records the shortest
offending call path. It never scores permissions. A known repository, CI, or
release mutation outside the grant is `REFUTED`; a missing indirect grant
without a known mutation is `UNKNOWN`; an unavailable external oracle is
`UNKNOWN`. Results reduce as `REFUTED > UNKNOWN > CLOSED`.

All generated output is written to a caller-owned directory. The CI corpus
contains a safe generator, a forbidden repository-write fixture, the two
indirect-grant distinctions, and an external-oracle case. Local validation is
not the evidence authority; the authoritative build, tests, conformance run,
and metrics come from GitHub Actions.

## Run in CI

```text
go run ./cmd/gooo-capability-effect-checker generate \
  --phase .gooo/capability-effect-checker.gooo \
  --corpus-root examples/corpus \
  --out /caller-owned/output \
  --source-root .
```

The generated report contains exact inferred effects, declared grants,
shortest offending paths, and the complete six-field UNKNOWN frontier. The
bootstrap lock is recorded in
[`contracts/bootstrap-lock-v1.json`](contracts/bootstrap-lock-v1.json), and
the protocol is described in [`docs/rfc-v1.md`](docs/rfc-v1.md).
