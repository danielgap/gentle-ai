# Archive Report: rdd-root-simplification-wave7

## Outcome

CLOSED — first native verification (2026-08-31) and archived. Wave landed 2026-08-04 via PRs #2437/#2438/#2439 (including its verify-remediation cycle).

## Final-state facts

- Exit checklist executed fresh, all six items with real evidence:
  1. Root module: 72 packages ok; sole failure is the environmental claude-mcp discovery test (introduced 2ff05014, 2026-08-10, post-wave and out of scope).
  2. Bench module build+vet+test exit 0.
  3. Driven bench with the CI-canonical invocation against a fresh `-trimpath` binary: portable corpus 61/61 completed (all 13 CI-pinned journeys), j105 completed, transition axis 11/11 completed, model-picker j97 untagged=unsupported / bench_fixture=completed.
  4. Deadcode ratchet holds exactly: 251-entry current set matches `.deadcode-baseline.txt` with 0 additions / 0 removals.
  5. Refusal ratchet tests green.
  6. gofmt empty, go vet clean.
- Tasks 18.1-18.6 were formally transferred to the successor `rdd-single-lifecycle-cutover` (documented in tasks.md); not wave7 outstanding work. Task completion 55/55 after the six-box reconciliation.
- Envelope: verdict `pass`, 14/14 requirements, 8/8 scenarios. Attempt settled `complete` (continued the same token across the inventory change caused by amending a stray file into its proper commit).

## Spec merge rulings

- MERGED (new capabilities): `rdd-single-lifecycle` (3 requirements — the switch-stays constraints wave7 delivered, including the full successor-transfer amendment context; review-mode switch verified alive in the CLI) and `rdd-legacy-retirement` (4 requirements — v1 freeze and legacy read retention verified alive: `legacy_v1_target_scoped_read_only` supported, deadcode ratchet green).
- SUPERSEDED (snapshot-only): `rdd-shadow-evaluation` — the living capability already carries wave7's requirements in evolved form (reworded 2026-08-21 by `3c3b8a02`); re-applying the older delta would regress newer text.

## Follow-up discovered during verification (non-blocking)

- `scripts/deadcode-ratchet.sh` pins `golang.org/x/tools/cmd/deadcode@v0.30.0`, which cannot analyze this go1.25.10 module set on this machine (export-data version error, silent exit 1 with `set -e`). The ratchet semantic passes under deadcode v0.49.0 (exact baseline match). Recommend bumping the pin — tracked here for a maintainer decision since CI may be silently affected or resolving differently.

## Warnings (non-blocking)

Same machine-local environmental claude-mcp failure as all of today's verifications; classified per the narrower-verification justification each time.
