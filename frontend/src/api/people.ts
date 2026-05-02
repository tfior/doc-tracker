import { get, post, patch, del, type ListResponse } from './client';

export interface Person {
  id: string;
  case_id: string;
  first_name: string;
  last_name: string;
  birth_date: string | null;
  birth_place: string | null;
  death_date: string | null;
  notes: string | null;
  parent_ids: string[];
  created_at: string;
  updated_at: string;
}

export interface UpdatePersonInput {
  first_name?: string;
  last_name?: string;
  birth_date?: string | null;
  birth_place?: string | null;
  death_date?: string | null;
  notes?: string | null;
}

export function listPeople(caseId: string): Promise<ListResponse<Person>> {
  return get<ListResponse<Person>>(`/cases/${caseId}/people`);
}

export function createPerson(caseId: string, input: { first_name: string; last_name: string } & UpdatePersonInput): Promise<Person> {
  return post<Person>(`/cases/${caseId}/people`, input);
}

export function updatePerson(caseId: string, personId: string, input: UpdatePersonInput): Promise<Person> {
  return patch<Person>(`/cases/${caseId}/people/${personId}`, input);
}

export function deletePerson(caseId: string, personId: string): Promise<void> {
  return del(`/cases/${caseId}/people/${personId}`);
}

export function addParent(caseId: string, personId: string, parentId: string): Promise<void> {
  return post<void>(`/cases/${caseId}/people/${personId}/relationships`, { parent_id: parentId });
}

export function removeParent(caseId: string, personId: string, parentId: string): Promise<void> {
  return del(`/cases/${caseId}/people/${personId}/relationships/${parentId}`);
}
