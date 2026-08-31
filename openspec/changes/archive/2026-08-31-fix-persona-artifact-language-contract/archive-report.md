# Archive Report: fix-persona-artifact-language-contract

## Outcome

CLOSED — verified PASS (with warnings) and archived 2026-08-31.

## Final-state facts (supersede stale snapshots in apply-progress/verify artifacts)

- The work landed on `main` as merge commit `b1735e24` — PR #2749 (58 files), 2026-08-08. The apply-progress claim "No commit, push, or PR was created" was an apply-time snapshot, superseded here.
- Verify executed 2026-08-31 against the landed tree: envelope `gentle-ai.verify-result/v1`, verdict `pass_with_warnings`, 9/9 requirements, 27/27 scenarios, blockers 0, critical 0. Evidence settled via `gentle-ai sdd-attempt` (settle state `complete`).
- Implementation verified present: all claimed test files resolve, all four spot-checked test names exist, `PersonaGentlemanNeutralArtifacts` exists in `internal/model/types.go` as a backward-compatible legacy alias.
- Spec capability merged to `openspec/specs/persona-artifact-language-contract/spec.md` (9 requirements, 27 scenarios, purely additive delta — no removals).

## Non-blocking warnings carried forward

1. Environmental test failure `TestInjectClaudeWorkspaceIsDiscoveredByNativeClaudeMCPList` (`internal/components/mcp`): introduced by commit `2ff05014` on 2026-08-10, two days after this change landed; failure depends on the local native `claude` CLI MCP project-scope discovery behavior. Unrelated to this change. Follow-up: investigate separately if full-suite green on this machine matters.
2. Engram cloud-sync doctor reports pending session mutations missing the `directory` field (`engram cloud upgrade doctor --project gentle-ai`); local Engram storage is functional.

## Deviations recorded at verify time

- OpenClaw and Trae markdown-section SDD injection route through `sddOrchestratorAsset(adapter.Agent())` and receive the generic SDD orchestrator instead of the Claude-specific asset (recorded in apply-progress; acceptable per design).
- Root and embedded `comment-writer` are behaviorally aligned, not byte-identical (enforced by contract tests, not diff equality).
