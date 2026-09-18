package reviewtransaction

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Terminal review receipts (#4405). The approved-acknowledge burn deletes the
// active compact authority, and before this store existed that deletion left
// no durable trace: negotiated STATUS and gate receipt discovery could not
// distinguish an acknowledged approval from a never-reviewed target for the
// same frozen candidate. A terminal receipt is the immutable, hashes-only
// record of what was approved and burned. It never carries review content,
// source bytes, or the acknowledgement token; it is not authority and grants
// nothing by itself.

const (
	terminalReceiptDirectoryName = "terminal-receipts"
	compactTerminalReceiptSchema = "gentle-ai/terminal-review-receipt/v1"
)

// TerminalReviewReceipt is the immutable record published when an approved
// compact acknowledgement burns its authority.
type TerminalReviewReceipt struct {
	Schema           string     `json:"schema"`
	LineageID        string     `json:"lineage_id"`
	TargetIdentity   string     `json:"target_identity"`
	ConsumedRevision string     `json:"consumed_revision"`
	Projection       Projection `json:"projection"`
	// CurrentSnapshot is the frozen snapshot the approval covered. It is
	// structural identity only (trees, digests, repository-relative paths) and
	// lets a gate compare exact scope without the live authority.
	CurrentSnapshot Snapshot  `json:"current_snapshot"`
	BurnedAt        time.Time `json:"burned_at"`
}

type terminalReceiptIndexEntry struct {
	ObjectRef string `json:"object_ref"`
}

func terminalReceiptObjectRef(receipt TerminalReviewReceipt) (string, error) {
	payload, err := json.Marshal(receipt)
	if err != nil {
		return "", fmt.Errorf("encode terminal review receipt: %w", err)
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

// publishCompactTerminalReviewReceipt durably writes the immutable receipt
// object and its by-target index entry. It is called before the burn deletes
// anything: a publication failure must leave the approved authority intact
// and inspectable.
func publishCompactTerminalReviewReceipt(base string, record CompactRecord, burnedAt time.Time) error {
	if record.State.State != StateApproved {
		return errors.New("terminal review receipt requires an approved compact authority") // refusal:by-design operator-knowledge: only an approved compact acknowledgement publishes a terminal receipt
	}
	if err := validateLineageID(record.State.LineageID); err != nil {
		return err
	}
	targetIdentity := record.State.CurrentSnapshot.Identity
	if !validSHA256(targetIdentity) {
		return errors.New("terminal review receipt requires a canonical target identity") // refusal:by-design operator-knowledge: the target identity comes from the approved authority record, never caller input
	}
	if !validSHA256(record.Revision) {
		return errors.New("terminal review receipt requires a canonical consumed revision") // refusal:by-design operator-knowledge: the consumed revision comes from the approved authority record, never caller input
	}
	receipt := TerminalReviewReceipt{
		Schema:           compactTerminalReceiptSchema,
		LineageID:        record.State.LineageID,
		TargetIdentity:   targetIdentity,
		ConsumedRevision: record.Revision,
		Projection:       record.State.CurrentSnapshot.Projection,
		CurrentSnapshot:  record.State.CurrentSnapshot,
		BurnedAt:         burnedAt.UTC(),
	}
	payload, err := json.Marshal(receipt)
	if err != nil {
		return fmt.Errorf("encode terminal review receipt: %w", err)
	}
	objectRef, err := terminalReceiptObjectRef(receipt)
	if err != nil {
		return err
	}
	indexEntry, err := json.Marshal(terminalReceiptIndexEntry{ObjectRef: objectRef})
	if err != nil {
		return fmt.Errorf("encode terminal review receipt index entry: %w", err)
	}

	root := filepath.Join(base, terminalReceiptDirectoryName, "v1")
	objectPath := filepath.Join(root, "objects", objectRef+".json")
	indexPath := filepath.Join(root, "by-target", targetIdentity, record.State.LineageID+".json")
	if err := ensurePrivateTerminalReceiptTree(base, filepath.Dir(objectPath)); err != nil {
		return fmt.Errorf("prepare terminal review receipt object store: %w", err)
	}
	if err := ensurePrivateTerminalReceiptTree(base, filepath.Dir(indexPath)); err != nil {
		return fmt.Errorf("prepare terminal review receipt index: %w", err)
	}
	// The immutable object is written first: an index entry may never point at
	// a missing object.
	if err := writeTerminalReceiptFileAtomic(objectPath, payload); err != nil {
		return fmt.Errorf("write terminal review receipt object: %w", err)
	}
	if err := writeTerminalReceiptFileAtomic(indexPath, indexEntry); err != nil {
		return fmt.Errorf("write terminal review receipt index entry: %w", err)
	}
	return nil
}

// ensurePrivateTerminalReceiptTree creates dir under the authority root with
// private permissions, refusing symlinked segments and syncing each parent on
// creation. Unlike the RAR walker it does not re-validate the shared authority
// root, which the compact store owns: only the receipt subtree is ours.
func ensurePrivateTerminalReceiptTree(authorityRoot, dir string) error {
	relative, err := filepath.Rel(authorityRoot, dir)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return errors.New("terminal receipt directory escapes authority root") // refusal:by-design human-authority: a receipt path escaping the authority store is tampering a maintainer must inspect
	}
	current := authorityRoot
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		if part == "" || part == "." || part == ".." {
			return errors.New("unsafe terminal receipt directory segment") // refusal:by-design human-authority: an unsafe receipt path segment is tampering a maintainer must inspect
		}
		parent := current
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if errors.Is(statErr, fs.ErrNotExist) {
			if err := os.Mkdir(current, 0o700); err != nil {
				return err
			}
			if err := SyncReviewDirectory(parent); err != nil {
				return err
			}
			info, statErr = os.Lstat(current)
		}
		if statErr != nil {
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("unsafe terminal receipt directory %q", current) // refusal:by-design human-authority: a symlinked or non-directory receipt path is tampering a maintainer must inspect
		}
	}
	return nil
}

// writeTerminalReceiptFileAtomic writes through a private temporary file in
// the destination directory and renames it into place, so readers never see a
// partial record.
func writeTerminalReceiptFileAtomic(path string, payload []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".terminal-receipt-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err := temporary.Write(payload); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, path)
}

// ResolveTerminalReviewReceiptsByTargetIdentity returns every terminal
// receipt whose acknowledged target identity exactly matches. Multiple
// entries are possible when the same frozen target was approved and
// acknowledged more than once across lineages.
func ResolveTerminalReviewReceiptsByTargetIdentity(base, targetIdentity string) ([]TerminalReviewReceipt, error) {
	indexPath := filepath.Join(base, terminalReceiptDirectoryName, "v1", "by-target", targetIdentity)
	entries, err := os.ReadDir(indexPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read terminal review receipt index: %w", err)
	}
	var receipts []TerminalReviewReceipt
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		indexBytes, err := os.ReadFile(filepath.Join(indexPath, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read terminal review receipt index entry: %w", err)
		}
		var index terminalReceiptIndexEntry
		if err := json.Unmarshal(indexBytes, &index); err != nil {
			return nil, fmt.Errorf("decode terminal review receipt index entry: %w", err)
		}
		objectBytes, err := os.ReadFile(filepath.Join(base, terminalReceiptDirectoryName, "v1", "objects", index.ObjectRef+".json"))
		if err != nil {
			return nil, fmt.Errorf("read terminal review receipt object %s: %w", index.ObjectRef, err)
		}
		var receipt TerminalReviewReceipt
		if err := json.Unmarshal(objectBytes, &receipt); err != nil {
			return nil, fmt.Errorf("decode terminal review receipt object %s: %w", index.ObjectRef, err)
		}
		if receipt.Schema != compactTerminalReceiptSchema {
			return nil, fmt.Errorf("terminal review receipt object %s has unknown schema %q", index.ObjectRef, receipt.Schema) // refusal:by-design operator-knowledge: an unknown receipt schema belongs to a different gentle-ai version
		}
		if receipt.TargetIdentity != targetIdentity {
			return nil, fmt.Errorf("terminal review receipt object %s indexes target %q under %q", index.ObjectRef, receipt.TargetIdentity, targetIdentity) // refusal:by-design human-authority: a receipt object contradicting its index is corruption a maintainer must inspect
		}
		receipts = append(receipts, receipt)
	}
	sort.Slice(receipts, func(i, j int) bool {
		if !receipts[i].BurnedAt.Equal(receipts[j].BurnedAt) {
			return receipts[i].BurnedAt.Before(receipts[j].BurnedAt)
		}
		return receipts[i].LineageID < receipts[j].LineageID
	})
	return receipts, nil
}

// LatestTerminalReviewReceiptByTargetIdentity returns the most recent
// terminal receipt for an exact frozen target identity.
func LatestTerminalReviewReceiptByTargetIdentity(base, targetIdentity string) (TerminalReviewReceipt, bool, error) {
	receipts, err := ResolveTerminalReviewReceiptsByTargetIdentity(base, targetIdentity)
	if err != nil {
		return TerminalReviewReceipt{}, false, err
	}
	if len(receipts) == 0 {
		return TerminalReviewReceipt{}, false, nil
	}
	return receipts[len(receipts)-1], true, nil
}

// ResolveLatestTerminalReviewReceipt resolves the latest terminal review
// receipt for a repository's exact frozen target identity (#4405). CLI and
// gate discovery route through this instead of the store layout.
func ResolveLatestTerminalReviewReceipt(ctx context.Context, repo, targetIdentity string) (TerminalReviewReceipt, bool, error) {
	root, _, err := reviewAuthorityRoot(ctx, repo)
	if err != nil {
		return TerminalReviewReceipt{}, false, err
	}
	return LatestTerminalReviewReceiptByTargetIdentity(root, targetIdentity)
}
