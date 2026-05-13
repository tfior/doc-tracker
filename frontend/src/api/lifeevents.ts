import { get, post, patch, del, type ListResponse } from './client';

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

export interface CreateLifeEventInput {
  person_id: string;
  event_type: string;
  event_date?: string | null;
  event_place?: string | null;
  spouse_name?: string | null;
  spouse_birth_date?: string | null;
  spouse_birth_place?: string | null;
  notes?: string | null;
}

export interface UpdateLifeEventInput {
  event_type?: string;
  event_date?: string | null;
  event_place?: string | null;
  spouse_name?: string | null;
  spouse_birth_date?: string | null;
  spouse_birth_place?: string | null;
  notes?: string | null;
}

export function listLifeEvents(caseId: string): Promise<ListResponse<LifeEvent>> {
  return get<ListResponse<LifeEvent>>(`/cases/${caseId}/life-events`);
}

export function createLifeEvent(caseId: string, input: CreateLifeEventInput): Promise<LifeEvent> {
  return post<LifeEvent>(`/cases/${caseId}/life-events`, input);
}

export function updateLifeEvent(caseId: string, eventId: string, input: UpdateLifeEventInput): Promise<LifeEvent> {
  return patch<LifeEvent>(`/cases/${caseId}/life-events/${eventId}`, input);
}

export function deleteLifeEvent(caseId: string, eventId: string): Promise<void> {
  return del(`/cases/${caseId}/life-events/${eventId}`);
}

export function reassignLifeEvent(caseId: string, eventId: string, personId: string): Promise<LifeEvent> {
  return patch<LifeEvent>(`/cases/${caseId}/life-events/${eventId}/person`, { person_id: personId });
}
