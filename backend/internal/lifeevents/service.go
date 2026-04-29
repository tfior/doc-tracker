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

func (s *Service) CreateLifeEvent(ctx context.Context, caseID, personID, eventType string, input UpdateLifeEventInput) (*LifeEvent, error) {
	return s.store.CreateLifeEvent(ctx, caseID, personID, eventType, input)
}

func (s *Service) UpdateLifeEvent(ctx context.Context, caseID, eventID string, input UpdateLifeEventInput) (*LifeEvent, error) {
	return s.store.UpdateLifeEvent(ctx, caseID, eventID, input)
}

func (s *Service) DeleteLifeEvent(ctx context.Context, caseID, eventID string) error {
	return s.store.DeleteLifeEvent(ctx, caseID, eventID)
}

func (s *Service) ReassignLifeEvent(ctx context.Context, caseID, eventID, personID string) (*LifeEvent, error) {
	return s.store.ReassignLifeEvent(ctx, caseID, eventID, personID)
}
