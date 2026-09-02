# v2 release-lineage plan

The v0.1.1 public release remains immutable and is recorded by release ID
`380149578`. Its annotated tag object, commit, CI metrics, run report, source
archive, release manifest, and SHA256SUMS are historical evidence.

The v2 change is PR-first: the pull request conformance job is green before
merge, then the same subject is checked on `main`. The first v0.1.2 release
attempt is preserved as a burned draft because its draft asset audit could not
use the tag lookup endpoint; the exact failure is recorded in
[`contracts/release-receipt-v0.1.2.json`](../contracts/release-receipt-v0.1.2.json).
The follow-up fix is PR-first as well, and the next fresh patch release is
`v0.1.3`. When a maintainer creates that tag, CI will:

1. verify the annotated tag and its merged-PR lineage;
2. verify the `.gooo` output-authority policy before any remote mutation;
3. rerun Go 1.27.0 verification and the 12-cell corpus in caller-owned temp;
4. create a draft release and verify every draft asset digest; and
5. publish once, then audit `immutable=true`, release ID, tag object, commit,
   and every public asset digest.

Existing public assets are never rewritten, and no local run is recorded as
authority. Improvement and external utility remain UNKNOWN without the
required evidence receipts.
