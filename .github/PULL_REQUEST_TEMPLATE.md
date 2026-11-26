## What and why

<!-- One paragraph. Link the issue or ADR when there is one. -->

## Contract impact

- [ ] No client-visible change (endpoint, tool, field, error code)
- [ ] Additive change (feature entry added to `internal/api/features.go` and `docs/contract.md`)
- [ ] Breaking change (contract version bumped, migration note written)

## Checks

- [ ] `go test ./...` passes
- [ ] `node scripts/check-skill-versions.mjs` passes (needed for any tool/error-code change)
- [ ] `pnpm --dir web typecheck` passes (needed for any `web/` change)
- [ ] Docs updated where behaviour is described

## Notes for the reviewer

<!-- Anything you are unsure about, or deliberately left out. -->
