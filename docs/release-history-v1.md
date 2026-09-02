# Release history

Release attempts are append-only evidence. A failed attempt is not deleted or
rewritten, and it does not become a product `CLOSED` result.

## Preserved v0.1.0 attempt

- annotated tag: `v0.1.0`
- tag object: `7b8e0a167870ffa72cf9c99dfc3ce4493ef02eb8`
- tag target: `18b21170de9befe22d40a86f4e97deabc405f949`
- Actions run: `33457327393`
- outcome: `UNKNOWN`
- stage: `release-preflight`
- step: `observe-immutable-releases`
- reason: `GitHub Actions GITHUB_TOKEN received HTTP 403 from the immutable-releases endpoint`
- unknown_class: `EXTERNAL_AUTHORITY`
- next_operation: `publish from a workflow that does not require the restricted setting read`
- blocked_by: `actions-token-admin-read`

The tag remains available as historical evidence. It is never reused. The
corrective release uses a new annotated `v0.1.1` tag and audits
`immutable=true` plus every published asset digest through the GitHub API.

## Read-only audit of preserved v0.1.1

- GitHub release ID: `380149578`; immutable: `true`
- annotated tag object: `a94e28c3bba56886b709838c04723e25abf55c1b`
- tag commit: `5bd2efb088ec9d38d3d4fc79c3545b61446935e8`
- release-manifest.json asset ID `538739362`, digest
  `sha256:f4a8373e9f5f466a7c77ca09866e59fa344ece00acdf515c5a8c2927d0c86bb9`
- ci-metrics.json asset ID `538739363`, digest
  `sha256:f2ef6d03150d868d85f8a3e91e6c34a47446f3417937339c291dac2447a61303`
- source archive asset ID `538739364`, digest
  `sha256:ca111a04da52d746223de7ce194433ac628bfc33754aea2655baa3a0e8b6ef2c`
- run-report.json asset ID `538739365`, digest
  `sha256:26089654c08e8db6353b253afb8a7c674aef6e813bd4d032545df0b973cd0843`
- SHA256SUMS asset ID `538739366`, digest
  `sha256:49938f8dd3dc3e6c5818f92cabad028a2a15e6678d76527c107c284122e19139`

The historical v1 phase digest is
`sha256:acc2147bdc1f6f9fee95a7f66ae79b955dfc6c701e145efb38fb735b7addf0fe`.
