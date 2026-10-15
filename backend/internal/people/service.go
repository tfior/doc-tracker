// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

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

func (s *Service) CreatePerson(ctx context.Context, caseID, firstName, lastName string, input UpdatePersonInput) (*Person, error) {
	return s.store.CreatePerson(ctx, caseID, firstName, lastName, input)
}

func (s *Service) UpdatePerson(ctx context.Context, caseID, personID string, input UpdatePersonInput) (*Person, error) {
	return s.store.UpdatePerson(ctx, caseID, personID, input)
}

func (s *Service) DeletePerson(ctx context.Context, caseID, personID string) error {
	return s.store.DeletePerson(ctx, caseID, personID)
}

func (s *Service) AddParent(ctx context.Context, caseID, personID, parentID string) (*Relationship, error) {
	return s.store.AddParent(ctx, caseID, personID, parentID)
}

func (s *Service) RemoveParent(ctx context.Context, caseID, personID, parentID string) error {
	return s.store.RemoveParent(ctx, caseID, personID, parentID)
}
