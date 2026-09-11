package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"hazop-safeguard-coverage/backend/internal/algorithm"
	"hazop-safeguard-coverage/backend/internal/dto"
	"hazop-safeguard-coverage/backend/internal/model"
	"hazop-safeguard-coverage/backend/internal/repository"
	"hazop-safeguard-coverage/backend/internal/util"
)

type conflictFixture struct {
	evaluations CoverageEvaluationService
	conflicts   IndependenceConflictService
	scenarioID  uint
	engineer    util.Actor
	reviewer    util.Actor
}

func newConflictFixture(t *testing.T) conflictFixture {
	t.Helper()
	db := testDB(t)
	nodeRepo := repository.NewProcessNodeRepository(db)
	scenarioRepo := repository.NewDeviationScenarioRepository(db)
	safeguardRepo := repository.NewSafeguardRepository(db)
	evaluationRepo := repository.NewCoverageEvaluationRepository(db)
	conflictRepo := repository.NewIndependenceConflictRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	now := time.Now().UTC().Truncate(time.Second)
	node := model.ProcessNode{
		NodeCode: "C-101", Name: "Conflict Node", UnitName: "Test Unit", Medium: "gas",
		DesignPressure: 1, DesignTemperature: 100, OwnerTeam: "test", Status: "active",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := nodeRepo.Create(context.Background(), &node); err != nil {
		t.Fatalf("create node: %v", err)
	}
	scenario := model.DeviationScenario{
		ProcessNodeID: node.ID, Guideword: "more", Parameter: "temperature",
		Cause: "cooling loss", Consequence: "overpressure", Likelihood: 4, Severity: 5,
		ScenarioState: "analyzed", Version: 1,
		CreatedBy: 10, CreatedByName: "engineer", CreatedAt: now, UpdatedAt: now,
	}
	if err := scenarioRepo.Create(context.Background(), &scenario); err != nil {
		t.Fatalf("create scenario: %v", err)
	}
	verified := now.AddDate(0, 0, -10)
	safeguards := []model.Safeguard{
		{Name: "SIS trip", SafeguardType: "interlock", TargetScenarioID: scenario.ID, IndependenceKey: "SIS-A", Effectiveness: 0.8, TestIntervalDays: 365, LastVerifiedAt: &verified, LifecycleState: "active", EvidenceNote: "cert 1", CreatedAt: now, UpdatedAt: now},
		{Name: "SIS alarm duplicate", SafeguardType: "alarm", TargetScenarioID: scenario.ID, IndependenceKey: "SIS-A", Effectiveness: 0.4, TestIntervalDays: 365, LastVerifiedAt: &verified, LifecycleState: "active", EvidenceNote: "shared final element", CreatedAt: now, UpdatedAt: now},
		{Name: "Relief valve", SafeguardType: "relief", TargetScenarioID: scenario.ID, IndependenceKey: "PSV-B", Effectiveness: 0.7, TestIntervalDays: 365, LastVerifiedAt: &verified, LifecycleState: "active", EvidenceNote: "cert 2", CreatedAt: now, UpdatedAt: now},
		{Name: "Backup relief", SafeguardType: "relief", TargetScenarioID: scenario.ID, IndependenceKey: "PSV-B", Effectiveness: 0.6, TestIntervalDays: 365, LastVerifiedAt: &verified, LifecycleState: "active", EvidenceNote: "same tap", CreatedAt: now, UpdatedAt: now},
	}
	for index := range safeguards {
		if err := safeguardRepo.Create(context.Background(), &safeguards[index]); err != nil {
			t.Fatalf("create safeguard: %v", err)
		}
	}
	return conflictFixture{
		evaluations: NewCoverageEvaluationService(evaluationRepo, scenarioRepo, nodeRepo, safeguardRepo, conflictRepo, auditRepo, algorithm.NewEvaluator()),
		conflicts:   NewIndependenceConflictService(conflictRepo, evaluationRepo, auditRepo),
		scenarioID:  scenario.ID,
		engineer:    util.Actor{UserID: 10, Username: "engineer", Role: "process_engineer", RequestID: "conflict-run"},
		reviewer:    util.Actor{UserID: 20, Username: "reviewer", Role: "safety_reviewer", RequestID: "conflict-review"},
	}
}

func (f conflictFixture) run(t *testing.T, key string) dto.CoverageEvaluationResponse {
	t.Helper()
	evaluation, duplicate, err := f.evaluations.Run(context.Background(), dto.RunCoverageEvaluationRequest{
		ScenarioID: f.scenarioID,
	}, key, f.engineer)
	if err != nil {
		t.Fatalf("run evaluation: %v", err)
	}
	if duplicate {
		t.Fatalf("run with fresh key %s returned a duplicate", key)
	}
	if evaluation.EvaluationState != "completed" {
		t.Fatalf("evaluation state = %s, want completed", evaluation.EvaluationState)
	}
	return evaluation
}

func conflictByKey(t *testing.T, items []dto.IndependenceConflictResponse, key string) dto.IndependenceConflictResponse {
	t.Helper()
	for _, item := range items {
		if item.IndependenceKey == key {
			return item
		}
	}
	t.Fatalf("conflict group %s not found in %#v", key, items)
	return dto.IndependenceConflictResponse{}
}

func TestIndependenceConflictReviewLifecycle(t *testing.T) {
	fixture := newConflictFixture(t)
	ctx := context.Background()
	evaluation := fixture.run(t, "conflict-run-0000001")

	list, err := fixture.conflicts.List(ctx, evaluation.ID)
	if err != nil {
		t.Fatalf("list conflicts: %v", err)
	}
	if list.PendingCount != 2 || len(list.Items) != 2 {
		t.Fatalf("pending = %d items = %d, want 2 pending groups", list.PendingCount, len(list.Items))
	}
	sisGroup := conflictByKey(t, list.Items, "SIS-A")
	if sisGroup.ResolutionState != "pending" || sisGroup.DedupReason == "" || len(sisGroup.IgnoredSafeguardIDs) != 1 {
		t.Fatalf("conflict group missing dedup basis: %#v", sisGroup)
	}
	if sisGroup.KeptSafeguardID == sisGroup.IgnoredSafeguardIDs[0] {
		t.Fatalf("kept and ignored safeguards must differ: %#v", sisGroup)
	}
	if len(list.PriorResolutions) != 0 {
		t.Fatalf("first evaluation must not have prior resolutions: %#v", list.PriorResolutions)
	}

	_, err = fixture.evaluations.Confirm(ctx, evaluation.ID, fixture.reviewer)
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Status != 409 || appErr.Code != util.CodeConflictPending {
		t.Fatalf("confirm with pending conflicts should be 409 INDEPENDENCE_CONFLICT_PENDING, got %v", err)
	}

	_, err = fixture.conflicts.Resolve(ctx, evaluation.ID, sisGroup.ID, dto.ResolveIndependenceConflictRequest{
		Action: "merge", Note: "invalid action",
	}, fixture.reviewer)
	if !errors.As(err, &appErr) || appErr.Status != 422 {
		t.Fatalf("invalid action should be 422, got %v", err)
	}
	_, err = fixture.conflicts.Resolve(ctx, evaluation.ID, sisGroup.ID, dto.ResolveIndependenceConflictRequest{
		Action: "accept", Note: "no",
	}, fixture.reviewer)
	if !errors.As(err, &appErr) || appErr.Status != 422 {
		t.Fatalf("short note should be 422, got %v", err)
	}
	_, err = fixture.conflicts.Resolve(ctx, evaluation.ID+999, sisGroup.ID, dto.ResolveIndependenceConflictRequest{
		Action: "accept", Note: "belongs to another evaluation",
	}, fixture.reviewer)
	if !errors.As(err, &appErr) || appErr.Status != 404 {
		t.Fatalf("conflict of another evaluation should be 404, got %v", err)
	}

	resolved, err := fixture.conflicts.Resolve(ctx, evaluation.ID, sisGroup.ID, dto.ResolveIndependenceConflictRequest{
		Action: "accept", Note: "同键报警与 SIS 共用最终元件，接受只计一次。",
	}, fixture.reviewer)
	if err != nil {
		t.Fatalf("resolve accept: %v", err)
	}
	if resolved.ResolutionState != "accepted" || resolved.ResolutionAction != "accept" ||
		resolved.ResolvedBy == nil || *resolved.ResolvedBy != fixture.reviewer.UserID || resolved.ResolvedAt == nil {
		t.Fatalf("unexpected resolution: %#v", resolved)
	}
	_, err = fixture.conflicts.Resolve(ctx, evaluation.ID, sisGroup.ID, dto.ResolveIndependenceConflictRequest{
		Action: "split", Note: "重复处理应被拒绝",
	}, fixture.reviewer)
	if !errors.As(err, &appErr) || appErr.Status != 409 {
		t.Fatalf("resolving twice should be 409, got %v", err)
	}

	_, err = fixture.evaluations.Confirm(ctx, evaluation.ID, fixture.reviewer)
	if !errors.As(err, &appErr) || appErr.Code != util.CodeConflictPending {
		t.Fatalf("one pending group should still block confirm, got %v", err)
	}

	psvGroup := conflictByKey(t, list.Items, "PSV-B")
	split, err := fixture.conflicts.Resolve(ctx, evaluation.ID, psvGroup.ID, dto.ResolveIndependenceConflictRequest{
		Action: "split", Note: "两个泄放阀取压点独立，要求拆分为不同独立性键。",
	}, fixture.reviewer)
	if err != nil || split.ResolutionState != "split" {
		t.Fatalf("resolve split: state=%s err=%v", split.ResolutionState, err)
	}
	confirmed, err := fixture.evaluations.Confirm(ctx, evaluation.ID, fixture.reviewer)
	if err != nil || confirmed.EvaluationState != "confirmed" {
		t.Fatalf("confirm after all conflicts resolved: state=%s err=%v", confirmed.EvaluationState, err)
	}

	voided, err := fixture.evaluations.Void(ctx, evaluation.ID, fixture.reviewer)
	if err != nil || voided.EvaluationState != "voided" {
		t.Fatalf("void confirmed evaluation: state=%s err=%v", voided.EvaluationState, err)
	}
	afterVoid, err := fixture.conflicts.List(ctx, evaluation.ID)
	if err != nil {
		t.Fatalf("list conflicts after void: %v", err)
	}
	if afterVoid.PendingCount != 0 || len(afterVoid.Items) != 2 {
		t.Fatalf("void must keep resolved conflict records: %#v", afterVoid.Items)
	}
	for _, item := range afterVoid.Items {
		if item.ResolutionNote == "" || item.ResolvedBy == nil {
			t.Fatalf("voided evaluation lost resolution data: %#v", item)
		}
	}
	_, err = fixture.conflicts.Resolve(ctx, evaluation.ID, afterVoid.Items[0].ID, dto.ResolveIndependenceConflictRequest{
		Action: "accept", Note: "作废后不允许再处理",
	}, fixture.reviewer)
	if !errors.As(err, &appErr) || appErr.Status != 409 {
		t.Fatalf("resolving on a voided evaluation should be 409, got %v", err)
	}

	rerun := fixture.run(t, "conflict-run-0000002")
	if rerun.ID == evaluation.ID {
		t.Fatal("re-run must create a new evaluation")
	}
	rerunList, err := fixture.conflicts.List(ctx, rerun.ID)
	if err != nil {
		t.Fatalf("list rerun conflicts: %v", err)
	}
	if rerunList.PendingCount != 2 {
		t.Fatalf("rerun pending = %d, want 2 fresh groups", rerunList.PendingCount)
	}
	if len(rerunList.PriorResolutions) != 2 {
		t.Fatalf("rerun should surface 2 retained prior resolutions, got %#v", rerunList.PriorResolutions)
	}
	priorSIS := conflictByKey(t, rerunList.PriorResolutions, "SIS-A")
	if priorSIS.EvaluationID != evaluation.ID || priorSIS.ResolutionState != "accepted" || priorSIS.ResolutionNote == "" {
		t.Fatalf("prior resolution not retained after void and rerun: %#v", priorSIS)
	}

	evidenced, err := fixture.conflicts.Resolve(ctx, rerun.ID, conflictByKey(t, rerunList.Items, "SIS-A").ID, dto.ResolveIndependenceConflictRequest{
		Action: "evidence", Note: "补充 SIL 验证报告 SIS-2026-118，证明共用元件。",
	}, fixture.reviewer)
	if err != nil || evidenced.ResolutionState != "evidenced" || evidenced.ResolutionAction != "evidence" {
		t.Fatalf("resolve evidence: state=%s err=%v", evidenced.ResolutionState, err)
	}
}

func TestEvaluationWithoutDuplicateKeysHasNoConflicts(t *testing.T) {
	db := testDB(t)
	nodeRepo := repository.NewProcessNodeRepository(db)
	scenarioRepo := repository.NewDeviationScenarioRepository(db)
	safeguardRepo := repository.NewSafeguardRepository(db)
	evaluationRepo := repository.NewCoverageEvaluationRepository(db)
	conflictRepo := repository.NewIndependenceConflictRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	now := time.Now().UTC().Truncate(time.Second)
	node := model.ProcessNode{
		NodeCode: "C-202", Name: "Clean Node", UnitName: "Test Unit", Medium: "water",
		DesignPressure: 1, DesignTemperature: 50, OwnerTeam: "test", Status: "active",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := nodeRepo.Create(context.Background(), &node); err != nil {
		t.Fatalf("create node: %v", err)
	}
	scenario := model.DeviationScenario{
		ProcessNodeID: node.ID, Guideword: "less", Parameter: "flow",
		Cause: "pump trip", Consequence: "loss of circulation", Likelihood: 2, Severity: 3,
		ScenarioState: "analyzed", Version: 1,
		CreatedBy: 10, CreatedByName: "engineer", CreatedAt: now, UpdatedAt: now,
	}
	if err := scenarioRepo.Create(context.Background(), &scenario); err != nil {
		t.Fatalf("create scenario: %v", err)
	}
	verified := now.AddDate(0, 0, -5)
	safeguard := model.Safeguard{
		Name: "Standby pump auto start", SafeguardType: "interlock", TargetScenarioID: scenario.ID,
		IndependenceKey: "PMP-STBY", Effectiveness: 0.7, TestIntervalDays: 180,
		LastVerifiedAt: &verified, LifecycleState: "active", EvidenceNote: "test record",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := safeguardRepo.Create(context.Background(), &safeguard); err != nil {
		t.Fatalf("create safeguard: %v", err)
	}
	evaluations := NewCoverageEvaluationService(evaluationRepo, scenarioRepo, nodeRepo, safeguardRepo, conflictRepo, auditRepo, algorithm.NewEvaluator())
	conflicts := NewIndependenceConflictService(conflictRepo, evaluationRepo, auditRepo)
	engineer := util.Actor{UserID: 10, Username: "engineer", Role: "process_engineer", RequestID: "clean-run"}
	reviewer := util.Actor{UserID: 20, Username: "reviewer", Role: "safety_reviewer", RequestID: "clean-review"}

	evaluation, _, err := evaluations.Run(context.Background(), dto.RunCoverageEvaluationRequest{
		ScenarioID: scenario.ID,
	}, "clean-run-00000001", engineer)
	if err != nil {
		t.Fatalf("run evaluation: %v", err)
	}
	list, err := conflicts.List(context.Background(), evaluation.ID)
	if err != nil {
		t.Fatalf("list conflicts: %v", err)
	}
	if list.PendingCount != 0 || len(list.Items) != 0 {
		t.Fatalf("evaluation without duplicate keys must have no conflicts: %#v", list.Items)
	}
	confirmed, err := evaluations.Confirm(context.Background(), evaluation.ID, reviewer)
	if err != nil || confirmed.EvaluationState != "confirmed" {
		t.Fatalf("confirm without conflicts should succeed: state=%s err=%v", confirmed.EvaluationState, err)
	}
}

func TestConflictListUnknownEvaluation(t *testing.T) {
	db := testDB(t)
	conflicts := NewIndependenceConflictService(
		repository.NewIndependenceConflictRepository(db),
		repository.NewCoverageEvaluationRepository(db),
		repository.NewAuditRepository(db),
	)
	_, err := conflicts.List(context.Background(), 4242)
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Status != 404 {
		t.Fatalf("unknown evaluation should be 404, got %v", err)
	}
}
