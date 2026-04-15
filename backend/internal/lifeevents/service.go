package lifeevents

import "context"

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListLifeEvents(ctx context.Context, caseID string, page, perPage int) ([]LifeEvent, int, error) {
	return s.store.ListLifeEvents(ctx, caseID, page, perPage)
}
