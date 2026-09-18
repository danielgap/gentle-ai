package reviewtransaction

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #4405: the approved-acknowledge burn must leave an immutable terminal
// receipt behind, keyed by the frozen target identity, so STATUS and gate
// discovery can distinguish an acknowledged approval from a never-reviewed
// target after the authority directory is deleted.
func TestAcknowledgeApprovedCompactAuthorityPublishesTerminalReceipt(t *testing.T) {
	repo, base, store, acknowledgement := approvedCompactAcknowledgementFixture(t, "terminal-receipt-publish")

	var record CompactRecord
	stateBytes, err := os.ReadFile(store.StatePath())
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stateBytes, &record); err != nil {
		t.Fatal(err)
	}

	if err := AcknowledgeApprovedCompactAuthority(context.Background(), repo, acknowledgement.LineageID, acknowledgement.TargetIdentity, acknowledgement.ExpectedRevision, acknowledgement.Token); err != nil {
		t.Fatal(err)
	}

	receipts, err := ResolveTerminalReviewReceiptsByTargetIdentity(base, acknowledgement.TargetIdentity)
	if err != nil {
		t.Fatal(err)
	}
	if len(receipts) != 1 {
		t.Fatalf("terminal receipts by target = %d, want 1", len(receipts))
	}
	receipt := receipts[0]
	if receipt.LineageID != acknowledgement.LineageID {
		t.Errorf("receipt lineage = %q, want %q", receipt.LineageID, acknowledgement.LineageID)
	}
	if receipt.TargetIdentity != acknowledgement.TargetIdentity {
		t.Errorf("receipt target identity = %q, want %q", receipt.TargetIdentity, acknowledgement.TargetIdentity)
	}
	if receipt.ConsumedRevision != acknowledgement.ExpectedRevision {
		t.Errorf("receipt consumed revision = %q, want %q", receipt.ConsumedRevision, acknowledgement.ExpectedRevision)
	}
	if receipt.Projection != record.State.CurrentSnapshot.Projection {
		t.Errorf("receipt projection = %q, want %q", receipt.Projection, record.State.CurrentSnapshot.Projection)
	}
	if receipt.CurrentSnapshot.Identity != acknowledgement.TargetIdentity {
		t.Errorf("receipt snapshot identity = %q, want %q", receipt.CurrentSnapshot.Identity, acknowledgement.TargetIdentity)
	}
	if receipt.BurnedAt.IsZero() {
		t.Error("receipt burn timestamp is zero")
	}
	if receipt.Schema != compactTerminalReceiptSchema {
		t.Errorf("receipt schema = %q, want %q", receipt.Schema, compactTerminalReceiptSchema)
	}
	// The acknowledgement token must never survive in the terminal receipt.
	receiptBytes, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if token := acknowledgement.Token; token != "" && strings.Contains(string(receiptBytes), token) {
		t.Error("terminal receipt leaks the acknowledgement token")
	}

	// The immutable object must exist and decode to the same receipt.
	entries, err := os.ReadDir(filepath.Join(base, terminalReceiptDirectoryName, "v1", "objects"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("terminal receipt objects = %d, want 1", len(entries))
	}
	objectBytes, err := os.ReadFile(filepath.Join(base, terminalReceiptDirectoryName, "v1", "objects", entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	var stored TerminalReviewReceipt
	if err := json.Unmarshal(objectBytes, &stored); err != nil {
		t.Fatal(err)
	}
	resolvedBytes, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if string(objectBytes) != string(resolvedBytes) {
		t.Errorf("stored object %s differs from resolved receipt", string(objectBytes))
	}
}

func TestResolveTerminalReviewReceiptByTargetIdentity(t *testing.T) {
	repo, base, _, acknowledgement := approvedCompactAcknowledgementFixture(t, "terminal-receipt-resolve")

	if err := AcknowledgeApprovedCompactAuthority(context.Background(), repo, acknowledgement.LineageID, acknowledgement.TargetIdentity, acknowledgement.ExpectedRevision, acknowledgement.Token); err != nil {
		t.Fatal(err)
	}

	latest, found, err := LatestTerminalReviewReceiptByTargetIdentity(base, acknowledgement.TargetIdentity)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("latest terminal receipt not found for the acknowledged target")
	}
	if latest.LineageID != acknowledgement.LineageID {
		t.Errorf("latest receipt lineage = %q, want %q", latest.LineageID, acknowledgement.LineageID)
	}

	_, found, err = LatestTerminalReviewReceiptByTargetIdentity(base, "sha256:"+strings.Repeat("00", 32))
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("terminal receipt resolved for a target identity that was never reviewed")
	}
}

// A failed receipt publication must abort the burn before any deletion: the
// approved authority stays inspectable and no by-target index entry appears.
func TestTerminalReceiptPublishFailureKeepsApprovedAuthority(t *testing.T) {
	repo, base, store, acknowledgement := approvedCompactAcknowledgementFixture(t, "terminal-receipt-publish-failure")

	// Block the immutable object directory with a regular file so publication
	// fails deterministically before any authority mutation.
	objectsPath := filepath.Join(base, terminalReceiptDirectoryName, "v1", "objects")
	if err := os.MkdirAll(filepath.Dir(objectsPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(objectsPath, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := AcknowledgeApprovedCompactAuthority(context.Background(), repo, acknowledgement.LineageID, acknowledgement.TargetIdentity, acknowledgement.ExpectedRevision, acknowledgement.Token)
	if err == nil {
		t.Fatal("acknowledge succeeded despite terminal receipt publication failure")
	}

	if _, statErr := os.Lstat(store.StatePath()); statErr != nil {
		t.Fatalf("approved authority was destroyed by a failed receipt publication: %v", statErr)
	}
	if _, statErr := os.Lstat(filepath.Join(base, terminalReceiptDirectoryName, "v1", "by-target", acknowledgement.TargetIdentity)); statErr == nil {
		t.Error("by-target index entry exists although publication failed")
	}
}
