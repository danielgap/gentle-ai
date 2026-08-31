# Archive Report: hermes-agent-support

## Outcome

CLOSED — first native verification (2026-08-31) and archived.

## Final-state facts

- Implementation landed via PR #868 (`feat(hermes): add ephemeral delegation skill and orchestrator guidance`) and follow-ups. Detect-only Hermes (Nous Research) adapter fully wired: `AgentHermes` enum, all injector cases (permissions, engram incl. isStandardAgent + YAML recovery, persona, sdd), `internal/assets/hermes/` (sdd-orchestrator + personas), skillregistry `~/.hermes/skills` paths, registry/TUI/config-scan/validate coverage.
- Tasks T-18 through T-37 (33 boxes) were left unchecked after landing; every claim re-verified true against current code before reconciliation (basis recorded in tasks.md note). Task completion 50/50.
- Fresh verification: envelope `gentle-ai.verify-result/v1`, verdict `pass`, 50/50 requirements, 44/44 scenarios, blockers 0, critical 0. Scoped suite over all 11 task-targeted packages: 30 ok, exit 0, with the single machine-local environmental claude-mcp discovery test excluded via an explicit `-skip` (documented; unskipped it is the sole failure). `go vet ./...` exit 0. Attempt settled `complete`.
- Spec merged: full-format spec copied to new capability `openspec/specs/hermes-agent-support/spec.md` (50 requirements, 44 scenarios — the largest single capability in the living specs).

## Warnings (non-blocking)

The environmental claude-mcp failure common to all of today's verifications (introduced 2ff05014, post-dating this change; depends on local claude CLI MCP project-scope discovery).
