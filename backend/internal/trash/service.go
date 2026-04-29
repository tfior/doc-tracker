package trash

import "context"

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) GetCaseTrash(ctx context.Context, caseID string) (*CaseTrash, error) {
	return s.store.GetCaseTrash(ctx, caseID)
}

func (s *Service) GetGlobalTrash(ctx context.Context) (*GlobalTrash, error) {
	return s.store.GetGlobalTrash(ctx)
}

func (s *Service) RestoreCase(ctx context.Context, caseID string) error {
	return s.store.RestoreCase(ctx, caseID)
}

func (s *Service) RestorePerson(ctx context.Context, caseID, personID string) error {
	return s.store.RestorePerson(ctx, caseID, personID)
}

func (s *Service) RestoreLifeEvent(ctx context.Context, caseID, eventID string) error {
	return s.store.RestoreLifeEvent(ctx, caseID, eventID)
}

func (s *Service) RestoreDocument(ctx context.Context, caseID, docID string) error {
	return s.store.RestoreDocument(ctx, caseID, docID)
}

func (s *Service) RestoreClaimLine(ctx context.Context, caseID, lineID string) error {
	return s.store.RestoreClaimLine(ctx, caseID, lineID)
}

func (s *Service) PermanentDeleteCase(ctx context.Context, caseID string) error {
	return s.store.PermanentDeleteCase(ctx, caseID)
}

func (s *Service) PermanentDeletePerson(ctx context.Context, caseID, personID string) error {
	return s.store.PermanentDeletePerson(ctx, caseID, personID)
}

func (s *Service) PermanentDeleteLifeEvent(ctx context.Context, caseID, eventID string) error {
	return s.store.PermanentDeleteLifeEvent(ctx, caseID, eventID)
}

func (s *Service) PermanentDeleteDocument(ctx context.Context, caseID, docID string) error {
	return s.store.PermanentDeleteDocument(ctx, caseID, docID)
}

func (s *Service) PermanentDeleteClaimLine(ctx context.Context, caseID, lineID string) error {
	return s.store.PermanentDeleteClaimLine(ctx, caseID, lineID)
}
