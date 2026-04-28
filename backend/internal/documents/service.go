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

func (s *Service) CreateDocument(ctx context.Context, caseID, personID, docType, title string, input UpdateDocumentInput) (*Document, error) {
	return s.store.CreateDocument(ctx, caseID, personID, docType, title, input)
}

func (s *Service) UpdateDocument(ctx context.Context, caseID, docID string, input UpdateDocumentInput) (*Document, error) {
	return s.store.UpdateDocument(ctx, caseID, docID, input)
}

func (s *Service) DeleteDocument(ctx context.Context, caseID, docID string) error {
	return s.store.DeleteDocument(ctx, caseID, docID)
}

func (s *Service) TransitionStatus(ctx context.Context, caseID, docID, statusKey string) (*Document, error) {
	return s.store.TransitionStatus(ctx, caseID, docID, statusKey)
}

func (s *Service) ReassignDocument(ctx context.Context, caseID, docID string, input ReassignDocumentInput) (*Document, error) {
	return s.store.ReassignDocument(ctx, caseID, docID, input)
}
