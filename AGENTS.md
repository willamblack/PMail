# PMail fork delivery policy

For implementation work in this repository, the owner requested this default
delivery order on 2026-09-22:

1. Verify changes, then commit and push `codex/admin-wildcard-catchall`.
2. After its checks pass, synchronize `master` without discarding independent changes.
3. Publish a new stable GitHub Release and the corresponding Docker image.
4. Verify release assets, Actions results, and GHCR manifests before reporting success.

Publish only to `willamblack/PMail` (normally remote `fork`), never upstream
`Jinnrry/PMail` (normally remote `origin`). Resolve remotes before pushing.
Do not force-push branches or overwrite old version tags. Preserve runtime data
and secrets; never deploy to the VPS automatically. A read-only request or a
review that needs no changes does not require a commit or release. Explicit
instructions for a particular task override this default. Details are in
`docs/RELEASING_CN.md`.
