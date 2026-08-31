# Archive Report: complete-native-review-lifecycle

## Outcome

CLOSED — verified PASS and archived 2026-08-31.

## Final-state facts

- Implementation landed via the native review lifecycle hardening work (`6bf8ff15` "fix(review): harden native review lifecycle recovery", 2026-07-16; change specs finalized `81dcdb26`, 2026-07-11).
- Verify envelope `gentle-ai.verify-result/v1` present and validated by native dispatcher (`archive: ready`, no blocked reasons).

## Supersession ruling (why deltas were NOT merged into the living spec)

The change's `specs/review-findings-ledger/spec.md` delta was finalized on or before 2026-07-11. The living capability spec `openspec/specs/review-findings-ledger/spec.md` was subsequently rewritten with newer semantics:

- `bb3256ab` (2026-07-18, "fix(spec): align findings-ledger lens fields with reviewer contract") rewrote `Initial findings freeze` — the delta's MODIFIED target — introducing `evidence_class`/`causal_disposition` and system-derived per-finding status. Applying the delta's older text would regress the living spec.
- The delta's ADDED requirement `Immutable genesis scope` (frozen canonical path set at genesis; corrections must not expand scope) is semantically covered by the living spec's evolved `Scope and incident boundaries` and `One ordinary correction transaction` requirements, which were later reworked through the 4R v2 precision-gating pass (`734a760b`) and the atomic lifecycle integration (`3c3b8a02`, 2026-08-21).

Decision: snapshot-only archive. The living spec is the authority maintained by later commits; re-applying stale deltas would corrupt it. Change dir preserved verbatim under `archive/2026-08-31-complete-native-review-lifecycle/`.

## Warnings (non-blocking)

None.
