package cases

import "context"

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListCases(ctx context.Context, page, perPage int) ([]Case, int, error) {
	return s.store.ListCases(ctx, page, perPage)
}

func (s *Service) GetCase(ctx context.Context, caseID string) (*CaseDetail, error) {
	return s.store.GetCase(ctx, caseID)
}
