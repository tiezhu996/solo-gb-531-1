package constants

type ConflictResolutionState string

const (
	ConflictPending   ConflictResolutionState = "pending"
	ConflictAccepted  ConflictResolutionState = "accepted"
	ConflictSplit     ConflictResolutionState = "split"
	ConflictEvidenced ConflictResolutionState = "evidenced"
)

const (
	ConflictActionAccept   = "accept"
	ConflictActionSplit    = "split"
	ConflictActionEvidence = "evidence"
)

var conflictActionStates = map[string]ConflictResolutionState{
	ConflictActionAccept:   ConflictAccepted,
	ConflictActionSplit:    ConflictSplit,
	ConflictActionEvidence: ConflictEvidenced,
}

func (s ConflictResolutionState) Valid() bool {
	switch s {
	case ConflictPending, ConflictAccepted, ConflictSplit, ConflictEvidenced:
		return true
	}
	return false
}

func ConflictStateForAction(action string) (ConflictResolutionState, bool) {
	state, ok := conflictActionStates[action]
	return state, ok
}

func ConflictResolutionStateValues() []string {
	return []string{"pending", "accepted", "split", "evidenced"}
}

func ConflictResolutionActionValues() []string {
	return []string{ConflictActionAccept, ConflictActionSplit, ConflictActionEvidence}
}
