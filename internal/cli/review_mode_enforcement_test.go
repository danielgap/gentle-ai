package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/reviewtransaction"
	"github.com/gentleman-programming/gentle-ai/v4/internal/state"
)

// TestReviewModeStatusReportsEnforcementFactsAdditively pins the S1 shape of
// the #1842 three-fact contract on the status surface: the enforcement
// projection grows the result envelope as its own object, the frozen
// gentle-ai.rdd-mode-status/v1 status object gains no field, and both new
// facts report absent until the slices that own them land.
func TestReviewModeStatusReportsEnforcementFactsAdditively(t *testing.T) {
	reviewModeHome(t)
	repo := initReviewCLIRepo(t)

	var output bytes.Buffer
	if err := RunReviewMode([]string{"status", "--cwd", repo, "--json"}, &output); err != nil {
		t.Fatalf("RunReviewMode(status --json) error = %v", err)
	}
	result := decodeReviewModeResult(t, output.Bytes())
	if result.Enforcement.Controller != reviewtransaction.RDDControllerAbsent ||
		result.Enforcement.DeliveryGate != reviewtransaction.RDDDeliveryGateAbsent ||
		result.Enforcement.Enforcement != reviewtransaction.RDDEnforcementAvailable {
		t.Fatalf("status enforcement projection = %#v, want absent controller, absent gate, available label", result.Enforcement)
	}
	// The status object stays exactly the object gentle-pi decodes; the
	// three-fact contract grows the envelope, never the frozen mode object.
	var envelope struct {
		Status map[string]json.RawMessage `json:"status"`
	}
	if err := json.Unmarshal(output.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	for key := range envelope.Status {
		switch key {
		case "schema", "global", "clone_local", "effective", "source", "revision", "reach":
		default:
			t.Fatalf("status envelope grew a field %q; gentle-pi decodes gentle-ai.rdd-mode-status/v1 as an exact object", key)
		}
	}

	output.Reset()
	if err := RunReviewMode([]string{"status", "--cwd", repo}, &output); err != nil {
		t.Fatalf("RunReviewMode(status) error = %v", err)
	}
	human := output.String()
	if !strings.Contains(human, "enforcement: available") ||
		!strings.Contains(human, "controller absent") ||
		!strings.Contains(human, "delivery gate absent") {
		t.Fatalf("human status omitted the enforcement facts:\n%s", human)
	}
}

// TestReviewModeEnforcementFollowsThePolicyFact pins that the enforcement
// projection follows the kill switch on every operation: while the switch is
// off the label reports off, and once it is on again the label reports the
// available floor. The new projection never decides anything on its own.
func TestReviewModeEnforcementFollowsThePolicyFact(t *testing.T) {
	home := reviewModeHome(t)
	repo := initReviewCLIRepo(t)

	var output bytes.Buffer
	if err := RunReviewMode([]string{"disable", "--scope", "global", "--cwd", repo, "--json"}, &output); err != nil {
		t.Fatalf("RunReviewMode(disable global) error = %v", err)
	}
	if result := decodeReviewModeResult(t, output.Bytes()); result.Enforcement.Enforcement != reviewtransaction.RDDEnforcementOff {
		t.Fatalf("disable enforcement label = %q, want off", result.Enforcement.Enforcement)
	}

	if err := state.Write(home, state.InstallState{RDDMode: string(reviewtransaction.RDDModeOn)}); err != nil {
		t.Fatalf("state.Write error = %v", err)
	}
	output.Reset()
	if err := RunReviewMode([]string{"status", "--cwd", repo}, &output); err != nil {
		t.Fatalf("RunReviewMode(status) error = %v", err)
	}
	if !strings.Contains(output.String(), "enforcement: available") {
		t.Fatalf("re-enabled status did not report the available floor:\n%s", output.String())
	}
}

// TestReviewModeStatusLeavesNoStateBehindWithEnforcement guards that reading
// the new projection did not smuggle a write into a read-only command.
func TestReviewModeStatusLeavesNoStateBehindWithEnforcement(t *testing.T) {
	reviewModeHome(t)
	repo := initReviewCLIRepo(t)

	var output bytes.Buffer
	if err := RunReviewMode([]string{"status", "--cwd", repo, "--json"}, &output); err != nil {
		t.Fatalf("RunReviewMode(status --json) error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(repo, ".git", "gentle-ai")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("status created repository state: %v", err)
	}
}
