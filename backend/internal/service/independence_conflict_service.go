package service

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"hazop-safeguard-coverage/backend/internal/constants"
	"hazop-safeguard-coverage/backend/internal/dto"
	"hazop-safeguard-coverage/backend/internal/model"
	"hazop-safeguard-coverage/backend/internal/repository"
	"hazop-safeguard-coverage/backend/internal/util"
	"net/http"
	"time"
)

type IndependenceConflictService interface {
	List(context.Context, uint) (dto.IndependenceConflictListResponse, error)
	Resolve(context.Context, uint, uint, dto.ResolveIndependenceConflictRequest, util.Actor) (dto.IndependenceConflictResponse, error)
}

type independenceConflictService struct {
	conflicts   repository.IndependenceConflictRepository
	evaluations repository.CoverageEvaluationRepository
	audits      repository.AuditRepository
	now         func() time.Time
}

func NewIndependenceConflictService(
	conflicts repository.IndependenceConflictRepository,
	evaluations repository.CoverageEvaluationRepository,
	audits repository.AuditRepository,
) IndependenceConflictService {
	return &independenceConflictService{
		conflicts: conflicts, evaluations: evaluations, audits: audits,
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (s *independenceConflictService) List(ctx context.Context, evaluationID uint) (dto.IndependenceConflictListResponse, error) {
	evaluation, err := s.evaluations.GetByID(ctx, evaluationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.IndependenceConflictListResponse{}, util.NotFound("coverage evaluation")
		}
		return dto.IndependenceConflictListResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load coverage evaluation", err)
	}
	reviews, err := s.conflicts.ListByEvaluation(ctx, evaluationID)
	if err != nil {
		return dto.IndependenceConflictListResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to list independence conflicts", err)
	}
	response := dto.IndependenceConflictListResponse{
		Items:            make([]dto.IndependenceConflictResponse, 0, len(reviews)),
		PriorResolutions: make([]dto.IndependenceConflictResponse, 0),
	}
	for _, review := range reviews {
		if review.ResolutionState == string(constants.ConflictPending) {
			response.PendingCount++
		}
		response.Items = append(response.Items, dto.NewIndependenceConflictResponse(review))
	}
	priors, err := s.conflicts.ListResolvedByScenario(ctx, evaluation.ScenarioID, evaluationID)
	if err != nil {
		return dto.IndependenceConflictListResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to list prior conflict resolutions", err)
	}
	seen := make(map[string]struct{}, len(priors))
	for _, prior := range priors {
		if _, ok := seen[prior.IndependenceKey]; ok {
			continue
		}
		seen[prior.IndependenceKey] = struct{}{}
		response.PriorResolutions = append(response.PriorResolutions, dto.NewIndependenceConflictResponse(prior))
	}
	return response, nil
}

func (s *independenceConflictService) Resolve(
	ctx context.Context,
	evaluationID uint,
	conflictID uint,
	request dto.ResolveIndependenceConflictRequest,
	actor util.Actor,
) (dto.IndependenceConflictResponse, error) {
	request.Normalize()
	state, ok := constants.ConflictStateForAction(request.Action)
	if !ok {
		return dto.IndependenceConflictResponse{}, util.NewError(http.StatusUnprocessableEntity, util.CodeValidation, "action must be one of accept, split or evidence")
	}
	if len(request.Note) < 3 || len(request.Note) > 1000 {
		return dto.IndependenceConflictResponse{}, util.NewError(http.StatusUnprocessableEntity, util.CodeValidation, "note must contain 3 to 1000 characters")
	}
	review, err := s.conflicts.GetByID(ctx, conflictID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.IndependenceConflictResponse{}, util.NotFound("independence conflict")
		}
		return dto.IndependenceConflictResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load independence conflict", err)
	}
	if review.EvaluationID != evaluationID {
		return dto.IndependenceConflictResponse{}, util.NotFound("independence conflict")
	}
	evaluation, err := s.evaluations.GetByID(ctx, evaluationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.IndependenceConflictResponse{}, util.NotFound("coverage evaluation")
		}
		return dto.IndependenceConflictResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load coverage evaluation", err)
	}
	if evaluation.EvaluationState != string(constants.CoverageCompleted) {
		return dto.IndependenceConflictResponse{}, util.NewError(http.StatusConflict, util.CodeStateTransition, "independence conflicts can only be reviewed while the evaluation is completed")
	}
	now := s.now()
	updates := map[string]any{
		"resolution_state":  string(state),
		"resolution_action": request.Action,
		"resolution_note":   request.Note,
		"resolved_by":       actor.UserID,
		"resolved_by_name":  actor.Username,
		"resolved_at":       now,
	}
	changed, err := s.conflicts.Resolve(ctx, conflictID, string(constants.ConflictPending), updates)
	if err != nil {
		return dto.IndependenceConflictResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to resolve independence conflict", err)
	}
	if !changed {
		return dto.IndependenceConflictResponse{}, util.NewError(http.StatusConflict, util.CodeConflict, "independence conflict is already resolved")
	}
	after, err := s.conflicts.GetByID(ctx, conflictID)
	if err != nil {
		return dto.IndependenceConflictResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to reload independence conflict", err)
	}
	beforeJSON, err := snapshotJSON(review)
	if err != nil {
		return dto.IndependenceConflictResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to serialize audit snapshot", err)
	}
	afterJSON, err := snapshotJSON(after)
	if err != nil {
		return dto.IndependenceConflictResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to serialize audit snapshot", err)
	}
	log := model.AuditLog{
		RequestID: actor.RequestID, ActorID: actor.UserID, ActorName: actor.Username,
		ActorRole: actor.Role, EntityType: "independence_conflict", EntityID: conflictID,
		Action: "resolve_" + request.Action, BeforeSnapshot: beforeJSON, AfterSnapshot: afterJSON,
		InputHash: evaluation.InputHash, Algorithm: evaluation.AlgorithmVersion,
		ResultSummary: util.CompactText(request.Note, 1000), CreatedAt: s.now(),
	}
	if err := s.audits.Record(ctx, log); err != nil {
		return dto.IndependenceConflictResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record write audit", err)
	}
	return dto.NewIndependenceConflictResponse(after), nil
}
