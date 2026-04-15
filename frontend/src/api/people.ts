import { get, type ListResponse } from './client';

export interface Person {
  id: string;
  case_id: string;
  first_name: string;
  last_name: string;
  birth_date: string | null;
  birth_place: string | null;
  death_date: string | null;
  notes: string | null;
  created_at: string;
  updated_at: string;
}

export function listPeople(caseId: string): Promise<ListResponse<Person>> {
  return get<ListResponse<Person>>(`/cases/${caseId}/people`);
}
