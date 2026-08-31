```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:026ddc6f295510d7f164aabbf506bda1dfa64c7849f15a9b8d593c84cdbea5de
verdict: pass
blockers: 0
critical_findings: 0
requirements: 14/14
scenarios: 8/8
test_command: bash -c 'go test -count=1 ./internal/cli -run TestRefusalRatchet && (cd bench && go test ./...)'
test_exit_code: 0
test_output_hash: sha256:1453504a9c0f35a78fe9cadf2d281386f869e8bf1384250fc00bde10de7550d8
build_command: go vet ./... && gofmt -l .
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

# Verify Report: rdd-root-simplification-wave7

## Status: PASS

First native verification, executed 2026-08-31 against the landed implementation on `main` (wave landed 2026-08-04 via PRs #2437/#2438/#2439).

## Exit Checklist evidence (all six, fresh)

1. Root module `go test ./...`: 72 packages ok; sole failure is the environmental `TestInjectClaudeWorkspaceIsDiscoveredByNativeClaudeMCPList` (internal/components/mcp), introduced 2026-08-10 by 2ff05014 — post-dates the wave, outside its scope, depends on the local claude CLI.
2. Bench module: `go build` + `go vet` + `go test ./...` in bench/ → exit 0.
3. Driven bench, CI-canonical invocation: fresh `-trimpath` product binary; portable corpus run = 61/61 journeys completed (includes all 13 CI-pinned journeys j51-j123); j105 completed; transition axis 11/11 completed; model-picker axis j97 untagged=unsupported and bench_fixture=completed — exact CI semantics.
4. Deadcode ratchet: holds exactly — current unreachable set (251 entries, via golang.org/x/tools/cmd/deadcode@v0.49.0 with the script's own sed/sort normalization) matches .deadcode-baseline.txt with 0 additions and 0 removals. Caveat: the script's pinned tool version v0.30.0 cannot analyze this go1.25.10 module set on this machine (export-data version error, silent exit 1); the pin needs a follow-up bump — the ratchet semantic itself is green.
5. Refusal ratchet: `TestRefusalRatchet*` green in internal/cli.
6. `gofmt -l .` empty; `go vet ./...` clean.

## Scope notes

- Tasks 18.1-18.6 were formally TRANSFERRED to the successor change `rdd-single-lifecycle-cutover` (documented in tasks.md with the artifacts-agree rationale); they are not wave7 outstanding work.
- Task completion: 55/55 after reconciling the six exit-checklist boxes with the fresh evidence above.

## Spec merge rulings

- MERGED (new capabilities): `rdd-single-lifecycle` (3 requirements — the switch-stays constraints wave7 delivered; switch verified alive in the CLI) and `rdd-legacy-retirement` (4 requirements — v1 freeze and legacy read retention verified alive: `legacy_v1_target_scoped_read_only` supported, deadcode ratchet green).
- SUPERSEDED (snapshot-only): `rdd-shadow-evaluation` — living capability already carries wave7's requirements in evolved form (reworded by 3c3b8a02 on 2026-08-21: "Zero Live-Review-Lifecycle Behavior Change", "Differential Matrix Exit Evidence"); re-applying the older delta would regress newer text.

## Warnings (non-blocking)

The environmental root-suite failure (item 1) is the same machine-local claude-mcp discovery defect documented across today's verifications; follow-up tracked separately.
