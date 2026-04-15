import { get, type ListResponse } from './client';

export interface LifeEvent {
  id: string;
  case_id: string;
  person_id: string;
  event_type: string;
  event_date: string | null;
  event_place: string | null;
  spouse_name: string | null;
  spouse_birth_date: string | null;
  spouse_birth_place: string | null;
  notes: string | null;
  has_documents: boolean;
  created_at: string;
  updated_at: string;
}

export function listLifeEvents(caseId: string): Promise<ListResponse<LifeEvent>> {
  return get<ListResponse<LifeEvent>>(`/cases/${caseId}/life-events`);
}
