# gentle-ai#1842 — managed/advisory RDD enforcement and controller contract

Issue: https://github.com/Gentleman-Programming/gentle-ai/issues/1842
Claim: issuecomment-5962229675 (2026-10-02, reporter edwinsaavedran credited;
approved design constraints by dnlrsls 2026-07-29)
Branch: feat/1842-mode-status-three-facts
Worktree: ~/gentleman/gentle-ai-1842 (base 5140c5f5)

## Approved design contract (binding)

- Three independent facts, never overwriting one another:
  policy (off|on, RDD mode store), controller (absent|declared|written|
  certified_current|stale|revoked|inconclusive|failed), delivery_gate
  (absent|advisory_current|enforced_current|stale|revoked|inconclusive|failed).
- Ownership: mode store owns policy; #1884 managed-resource identity owns
  installed controller/gate extents (ALREADY MERGED); #1873 owns runtime
  certification records (IN FLIGHT as PR #2923 by jjeg1979 — do not touch).
- Fail-closed derivation of public label: off|available|advisory|managed.
- Codex stays `available` until a trustworthy loader oracle exists.
- `managed` reserved for an exact non-bypassable repository/ref boundary proven
  by positive and negative native tests.

## Slice plan (each its own PR)

- [ ] S1 — `review mode status` three-fact contract + fail-closed label
  derivation. No new sources of truth; certification read opportunistically;
  absent-certified never yields `managed`.
- [ ] S2 — compact always-loaded controller contract per supported agent,
  independent of optional SDD.
- [ ] S3 — certification wiring after #2923 merges (promotes derivations).
- [ ] S4 — `managed` delivery boundary: non-bypassable repo/ref gate with
  positive and negative native tests.
- [ ] S5 — per-adapter end-to-end conformance tests (receipt or explicit
  advisory/available, never silent managed).

## Evidence log

- 2026-10-02 claim posted with dependency state (#1884 merged, #2923 in
  flight, #2695 is unrelated MCP work that only references #1873).
- 2026-10-03 S1 implemented: `internal/reviewtransaction/rdd_enforcement.go`
  (fact vocabulary + pure `DeriveEnforcement` + `ResolveRDDEnforcement` with
  structurally absent facts), matrix tests in `rdd_enforcement_test.go`, and
  additive emission in `internal/cli/review_mode.go` (top-level `enforcement`
  object; the frozen `gentle-ai.rdd-mode-status/v1` status object gains no
  field — pinned by test). #1884 extents surface located: `internal/pathidentity`.
  Full `internal/reviewtransaction` + `internal/cli` suites green; gofmt/vet
  clean. RED observed first (build failure on missing types). Work-unit
  commit: b4f08450.

## Exploration map (2026-10-02 night)

- Policy fact: `internal/reviewtransaction/rdd_mode.go` — `RDDModeStatus`
  {Schema, Global, CloneLocal, Effective, Source, Revision, Reach}; resolved by
  `reviewModeStatus(ctx, repo)` in `internal/cli/review_mode.go:120` (global +
  clone-local, off-only override semantics, CAS revision).
- Delivery projection that already exists: `RDDDelivery` enum in rdd_mode.go
  (receipt_governed / disabled-unmanaged / unmanaged /
  candidate_declined-unmanaged) — none of its values is an approval.
- Emission: `emitReviewMode(stdout, result, emitJSON)` in review_mode.go — S1
  adds fields ADDITIVELY (controller, delivery_gate, enforcement); existing
  consumers (TUI screens, stop hook, gentle-pi TS reader) must not break.
- Fail-closed precedent: `reviewDrivenDevelopmentDisabled` (unreadable switch
  fails closed to enabled); `RDDModeReach` documents the #3284 partial-write
  lesson — the controller/gate facts must follow the same honesty discipline.
- #1884 machinery lives in the doctor mixed-asset detection area (locate
  `grep -rn "mixed" internal/doctor` next session for the extents reuse).
- #2923 (jjeg1979 runtime oracle): do not touch; S1 treats certification as
  structurally absent so `managed` is unreachable by construction in S1.

## S1 design decision

New `RDDEnforcementStatus` in reviewtransaction: controller + delivery_gate
states per the approved model, with a pure `DeriveEnforcement(policy,
controller, gate) -> off|available|advisory|managed` function. Unit tests
cover the full fact matrix (including tampered/stale/inconclusive → fail
closed, never managed). CLI status emits the new fields additively. RED first:
matrix tests fail on the missing type, then implement.

## Resume point

Next session: native review of S1 is BLOCKED by a gentle-shell/gentle-pi
retained-route defect (details in the 2026-10-03 incident note below):
forecast accepted, every acknowledgement rejected as `capture-binding-rejected`
("missing or stale") while both facade STATUS and a byte-exact CLI replica of
the tool's query still offer the same binding; an explicit-input STATUS then
fails deterministically with `capture-route-registration-rejected` (route
collision between baseRef spellings: ref name `upstream/main` at START vs
resolved `28b6bc3f…` tree afterwards). Lineage review-fabf4abe0b1dfa98
stays in `reviewing`; nothing burned, nothing fabricated. Next: retry capture
after a pi session restart (retained maps are in-memory) or after the stack
defect is fixed; then acknowledge-approved → branch-pr.

## S1 delivery record (2026-10-03)

Native review COMPLETED after the pi restart cleared the in-memory retained
maps: fresh lineage review-6a7097b9aa471773 (committed range 5140c5f5..7d2d560b,
533 lines) approved and acknowledged, authority burned
(gentle-ai.review-acknowledged/v1). Advisory findings left as follow-ups:
error-path enforcement unproved (review_mode.go:99-103, WARNING), enforcement
JSON keys unpinned (test), enforcement object without schema version.

Delivery split into a 2-PR chain (533 > 400 budget, user decision):
PR #5211 = slice 1 contract (320 lines, deadcode baseline +4 pending its
caller); slice 2 = CLI emission wiring + tests + this doc (branch
feat/1842-mode-status-enforcement-emission, opens after #5211 merges; GitHub
cannot use a fork branch as a PR base). Baseline tightened back in slice 2
(all four functions reachable again). Advisory findings deferred to the S2+
slices where the facts gain real sources.
