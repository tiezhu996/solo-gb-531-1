package dto

import (
	"encoding/json"
	"hazop-safeguard-coverage/backend/internal/model"
	"strings"
	"time"
)

type ResolveIndependenceConflictRequest struct {
	Action string `json:"action" binding:"required"`
	Note   string `json:"note" binding:"required"`
}

func (r *ResolveIndependenceConflictRequest) Normalize() {
	r.Action = strings.ToLower(strings.TrimSpace(r.Action))
	r.Note = strings.TrimSpace(r.Note)
}

type IndependenceConflictResponse struct {
	ID                  uint       `json:"id"`
	EvaluationID        uint       `json:"evaluation_id"`
	ScenarioID          uint       `json:"scenario_id"`
	IndependenceKey     string     `json:"independence_key"`
	KeptSafeguardID     uint       `json:"kept_safeguard_id"`
	IgnoredSafeguardIDs []uint     `json:"ignored_safeguard_ids"`
	DedupReason         string     `json:"dedup_reason"`
	ResolutionState     string     `json:"resolution_state"`
	ResolutionAction    string     `json:"resolution_action,omitempty"`
	ResolutionNote      string     `json:"resolution_note,omitempty"`
	ResolvedBy          *uint      `json:"resolved_by,omitempty"`
	ResolvedByName      string     `json:"resolved_by_name,omitempty"`
	ResolvedAt          *time.Time `json:"resolved_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

type IndependenceConflictListResponse struct {
	Items            []IndependenceConflictResponse `json:"items"`
	PendingCount     int                            `json:"pending_count"`
	PriorResolutions []IndependenceConflictResponse `json:"prior_resolutions"`
}

func NewIndependenceConflictResponse(review model.IndependenceConflictReview) IndependenceConflictResponse {
	ignored := make([]uint, 0)
	_ = json.Unmarshal([]byte(review.IgnoredSafeguardIDs), &ignored)
	return IndependenceConflictResponse{
		ID: review.ID, EvaluationID: review.EvaluationID, ScenarioID: review.ScenarioID,
		IndependenceKey: review.IndependenceKey, KeptSafeguardID: review.KeptSafeguardID,
		IgnoredSafeguardIDs: ignored, DedupReason: review.DedupReason,
		ResolutionState: review.ResolutionState, ResolutionAction: review.ResolutionAction,
		ResolutionNote: review.ResolutionNote, ResolvedBy: review.ResolvedBy,
		ResolvedByName: review.ResolvedByName, ResolvedAt: review.ResolvedAt,
		CreatedAt: review.CreatedAt,
	}
}
