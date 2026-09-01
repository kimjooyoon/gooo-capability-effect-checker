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
