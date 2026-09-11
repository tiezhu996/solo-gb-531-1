package repository

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"hazop-safeguard-coverage/backend/internal/constants"
	"hazop-safeguard-coverage/backend/internal/model"
	"time"
)

type IndependenceConflictRepository interface {
	CreateBatch(context.Context, []model.IndependenceConflictReview) error
	ListByEvaluation(context.Context, uint) ([]model.IndependenceConflictReview, error)
	GetByID(context.Context, uint) (model.IndependenceConflictReview, error)
	CountPendingByEvaluation(context.Context, uint) (int64, error)
	Resolve(context.Context, uint, string, map[string]any) (bool, error)
	ListResolvedByScenario(context.Context, uint, uint) ([]model.IndependenceConflictReview, error)
}

type independenceConflictRepository struct{ db *gorm.DB }

func NewIndependenceConflictRepository(db *gorm.DB) IndependenceConflictRepository {
	return &independenceConflictRepository{db: db}
}

func (r *independenceConflictRepository) CreateBatch(ctx context.Context, reviews []model.IndependenceConflictReview) error {
	if len(reviews) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Create(&reviews).Error; err != nil {
		return fmt.Errorf("create independence conflict reviews: %w", err)
	}
	return nil
}

func (r *independenceConflictRepository) ListByEvaluation(ctx context.Context, evaluationID uint) ([]model.IndependenceConflictReview, error) {
	var reviews []model.IndependenceConflictReview
	if err := r.db.WithContext(ctx).Where("evaluation_id = ?", evaluationID).
		Order("independence_key ASC, id ASC").Find(&reviews).Error; err != nil {
		return nil, fmt.Errorf("list independence conflicts for evaluation %d: %w", evaluationID, err)
	}
	return reviews, nil
}

func (r *independenceConflictRepository) GetByID(ctx context.Context, id uint) (model.IndependenceConflictReview, error) {
	var review model.IndependenceConflictReview
	if err := r.db.WithContext(ctx).First(&review, id).Error; err != nil {
		return model.IndependenceConflictReview{}, fmt.Errorf("find independence conflict %d: %w", id, err)
	}
	return review, nil
}

func (r *independenceConflictRepository) CountPendingByEvaluation(ctx context.Context, evaluationID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.IndependenceConflictReview{}).
		Where("evaluation_id = ? AND resolution_state = ?", evaluationID, string(constants.ConflictPending)).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count pending independence conflicts for evaluation %d: %w", evaluationID, err)
	}
	return count, nil
}

func (r *independenceConflictRepository) Resolve(ctx context.Context, id uint, expectedState string, updates map[string]any) (bool, error) {
	values := make(map[string]any, len(updates)+1)
	for key, value := range updates {
		values[key] = value
	}
	values["updated_at"] = time.Now().UTC()
	result := r.db.WithContext(ctx).Model(&model.IndependenceConflictReview{}).
		Where("id = ? AND resolution_state = ?", id, expectedState).Updates(values)
	if result.Error != nil {
		return false, fmt.Errorf("resolve independence conflict %d: %w", id, result.Error)
	}
	return result.RowsAffected == 1, nil
}

func (r *independenceConflictRepository) ListResolvedByScenario(ctx context.Context, scenarioID uint, excludeEvaluationID uint) ([]model.IndependenceConflictReview, error) {
	var reviews []model.IndependenceConflictReview
	if err := r.db.WithContext(ctx).
		Where("scenario_id = ? AND evaluation_id != ? AND resolution_state != ?",
			scenarioID, excludeEvaluationID, string(constants.ConflictPending)).
		Order("resolved_at DESC, id DESC").Find(&reviews).Error; err != nil {
		return nil, fmt.Errorf("list resolved independence conflicts for scenario %d: %w", scenarioID, err)
	}
	return reviews, nil
}
