package model

import "time"

type IndependenceConflictReview struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	EvaluationID        uint       `gorm:"not null;index:idx_conflict_scenario_evaluation" json:"evaluation_id"`
	ScenarioID          uint       `gorm:"not null;index:idx_conflict_scenario_evaluation" json:"scenario_id"`
	IndependenceKey     string     `gorm:"size:100;not null" json:"independence_key"`
	KeptSafeguardID     uint       `gorm:"not null" json:"kept_safeguard_id"`
	IgnoredSafeguardIDs string     `gorm:"type:text;not null" json:"ignored_safeguard_ids"`
	DedupReason         string     `gorm:"type:text;not null" json:"dedup_reason"`
	ResolutionState     string     `gorm:"size:24;not null;index" json:"resolution_state"`
	ResolutionAction    string     `gorm:"size:24" json:"resolution_action,omitempty"`
	ResolutionNote      string     `gorm:"type:text" json:"resolution_note,omitempty"`
	ResolvedBy          *uint      `json:"resolved_by,omitempty"`
	ResolvedByName      string     `gorm:"size:80" json:"resolved_by_name,omitempty"`
	ResolvedAt          *time.Time `json:"resolved_at,omitempty"`
	CreatedAt           time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"not null" json:"updated_at"`
}

func (IndependenceConflictReview) TableName() string { return "independence_conflict_reviews" }
