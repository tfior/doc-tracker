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
