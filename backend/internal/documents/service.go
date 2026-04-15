package documents

import "context"

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListDocuments(ctx context.Context, caseID string, page, perPage int) ([]Document, int, error) {
	return s.store.ListDocuments(ctx, caseID, page, perPage)
}
