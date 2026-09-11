package constants

import "testing"

func TestConflictStateForAction(t *testing.T) {
	t.Parallel()
	tests := []struct {
		action string
		state  ConflictResolutionState
		ok     bool
	}{
		{ConflictActionAccept, ConflictAccepted, true},
		{ConflictActionSplit, ConflictSplit, true},
		{ConflictActionEvidence, ConflictEvidenced, true},
		{"merge", "", false},
		{"", "", false},
	}
	for _, test := range tests {
		state, ok := ConflictStateForAction(test.action)
		if state != test.state || ok != test.ok {
			t.Errorf("action %q: got (%q, %t), want (%q, %t)", test.action, state, ok, test.state, test.ok)
		}
	}
}

func TestConflictResolutionStateValid(t *testing.T) {
	t.Parallel()
	for _, value := range ConflictResolutionStateValues() {
		if !ConflictResolutionState(value).Valid() {
			t.Errorf("state %q should be valid", value)
		}
	}
	if ConflictResolutionState("reopened").Valid() {
		t.Error("unknown state must not be valid")
	}
}
