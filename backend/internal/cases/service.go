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

func (s *Service) CreateCase(ctx context.Context, title string) (*Case, error) {
	return s.store.CreateCase(ctx, title)
}

func (s *Service) UpdateCase(ctx context.Context, caseID string, title *string, status *string) (*Case, error) {
	return s.store.UpdateCase(ctx, caseID, title, status)
}

func (s *Service) DeleteCase(ctx context.Context, caseID string) error {
	return s.store.DeleteCase(ctx, caseID)
}
