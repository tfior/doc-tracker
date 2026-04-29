package activitylog

import "context"

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// Insert records an activity log entry. Errors are non-fatal from the caller's
// perspective — the underlying operation has already succeeded.
func (s *Service) Insert(ctx context.Context, p InsertParams) error {
	return s.store.Insert(ctx, p)
}
