package service

import (
	"context"
	"testing"
	"time"
)

type costReviewRepoFake struct {
	approved     bool
	input        SupplierProviderCostReviewApproveInput
	approvedMany bool
	bulkInput    SupplierProviderCostReviewBulkApproveInput
}

func (f *costReviewRepoFake) List(context.Context, SupplierProviderCostReviewListParams) (SupplierProviderCostReviewListResult, error) {
	return SupplierProviderCostReviewListResult{}, nil
}
func (f *costReviewRepoFake) History(context.Context, int64) ([]SupplierProviderCostReviewHistory, error) {
	return nil, nil
}
func (f *costReviewRepoFake) Sync(context.Context, SupplierProviderCostReviewSyncInput) (*SupplierProviderCostReview, error) {
	return nil, nil
}
func (f *costReviewRepoFake) Approve(_ context.Context, id int64, input SupplierProviderCostReviewApproveInput) (*SupplierProviderCostReview, error) {
	f.approved = id == 7
	f.input = input
	return &SupplierProviderCostReview{ID: id, FinalCost: input.ManualCost}, nil
}
func (f *costReviewRepoFake) ApproveMany(_ context.Context, input SupplierProviderCostReviewBulkApproveInput) ([]SupplierProviderCostReview, error) {
	f.approvedMany = true
	f.bulkInput = input
	return []SupplierProviderCostReview{{ID: input.Items[0].ID}}, nil
}

func TestSupplierProviderCostReviewServiceApproveManualValidatesAndDelegates(t *testing.T) {
	repo := &costReviewRepoFake{}
	svc := NewSupplierProviderCostReviewService(repo)
	got, err := svc.Approve(context.Background(), 7, SupplierProviderCostReviewApproveInput{DecisionType: CostReviewDecisionManual, ManualCost: ptrFloat(12.345678), Version: 2, OperatorID: 9})
	if err != nil {
		t.Fatal(err)
	}
	if !repo.approved || got.FinalCost == nil || *got.FinalCost != 12.345678 {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestSupplierProviderCostReviewServiceApproveRejectsInvalidManualCost(t *testing.T) {
	svc := NewSupplierProviderCostReviewService(&costReviewRepoFake{})
	for _, cost := range []float64{-1, 1.1234567} {
		if _, err := svc.Approve(context.Background(), 7, SupplierProviderCostReviewApproveInput{DecisionType: CostReviewDecisionManual, ManualCost: &cost}); err == nil {
			t.Fatalf("expected invalid cost %v", cost)
		}
	}
}

func TestSupplierProviderCostReviewServiceApproveManyRejectsEmptyItems(t *testing.T) {
	svc := NewSupplierProviderCostReviewService(&costReviewRepoFake{})
	_, err := svc.ApproveMany(context.Background(), SupplierProviderCostReviewBulkApproveInput{DecisionType: CostReviewDecisionCalculated})
	if err == nil {
		t.Fatal("expected empty batch to fail")
	}
}

func TestSupplierProviderCostReviewServiceApproveManyRejectsDuplicateIDs(t *testing.T) {
	svc := NewSupplierProviderCostReviewService(&costReviewRepoFake{})
	_, err := svc.ApproveMany(context.Background(), SupplierProviderCostReviewBulkApproveInput{
		Items:        []SupplierProviderCostReviewApproveItem{{ID: 7, Version: 1}, {ID: 7, Version: 2}},
		DecisionType: CostReviewDecisionCalculated,
	})
	if err == nil {
		t.Fatal("expected duplicate IDs to fail")
	}
}

func TestSupplierProviderCostReviewServiceApproveManyRejectsInvalidDecision(t *testing.T) {
	svc := NewSupplierProviderCostReviewService(&costReviewRepoFake{})
	_, err := svc.ApproveMany(context.Background(), SupplierProviderCostReviewBulkApproveInput{
		Items:        []SupplierProviderCostReviewApproveItem{{ID: 7, Version: 1}},
		DecisionType: "invalid",
	})
	if err == nil {
		t.Fatal("expected invalid decision to fail")
	}
}

func TestSupplierProviderCostReviewServiceApproveManyRejectsInvalidManualCost(t *testing.T) {
	svc := NewSupplierProviderCostReviewService(&costReviewRepoFake{})
	for _, cost := range []*float64{nil, ptrFloat(1.1234567)} {
		_, err := svc.ApproveMany(context.Background(), SupplierProviderCostReviewBulkApproveInput{
			Items:        []SupplierProviderCostReviewApproveItem{{ID: 7, Version: 1}},
			DecisionType: CostReviewDecisionManual,
			ManualCost:   cost,
		})
		if err == nil {
			t.Fatalf("expected invalid manual cost %#v", cost)
		}
	}
}

func TestSupplierProviderCostReviewServiceApproveManyValidatesAndDelegates(t *testing.T) {
	repo := &costReviewRepoFake{}
	svc := NewSupplierProviderCostReviewService(repo)
	items := []SupplierProviderCostReviewApproveItem{{ID: 7, Version: 2}, {ID: 8, Version: 3}}
	got, err := svc.ApproveMany(context.Background(), SupplierProviderCostReviewBulkApproveInput{
		Items: items, DecisionType: CostReviewDecisionManual, ManualCost: ptrFloat(12.345678), OperatorID: 9,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !repo.approvedMany || len(got) != 1 || len(repo.bulkInput.Items) != 2 || repo.bulkInput.OperatorID != 9 {
		t.Fatalf("unexpected result: %#v %#v", got, repo.bulkInput)
	}
}

func ptrFloat(v float64) *float64 { return &v }

var _ = time.Time{}
