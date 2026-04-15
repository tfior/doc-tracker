package people

import "context"

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListPeople(ctx context.Context, caseID string, page, perPage int) ([]Person, int, error) {
	return s.store.ListPeople(ctx, caseID, page, perPage)
}
