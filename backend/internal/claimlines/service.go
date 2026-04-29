package claimlines

import "context"

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListClaimLines(ctx context.Context, caseID string, page, perPage int) ([]ClaimLine, int, error) {
	return s.store.ListClaimLines(ctx, caseID, page, perPage)
}

func (s *Service) CreateClaimLine(ctx context.Context, caseID, rootPersonID, status string, notes NullableField) (*ClaimLine, error) {
	return s.store.CreateClaimLine(ctx, caseID, rootPersonID, status, notes)
}

func (s *Service) UpdateClaimLine(ctx context.Context, caseID, lineID string, status *string, notes NullableField) (*ClaimLine, error) {
	return s.store.UpdateClaimLine(ctx, caseID, lineID, status, notes)
}

func (s *Service) DeleteClaimLine(ctx context.Context, caseID, lineID string) error {
	return s.store.DeleteClaimLine(ctx, caseID, lineID)
}
