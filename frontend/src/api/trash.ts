import { get, post, del } from './client';

export interface TrashedCase {
  id: string;
  title: string;
  status: string;
  deleted_at: string;
}

export interface GlobalTrash {
  cases: TrashedCase[];
}

export interface TrashedPerson {
  id: string;
  case_id: string;
  first_name: string;
  last_name: string;
  deleted_at: string;
}

export interface TrashedLifeEvent {
  id: string;
  case_id: string;
  person_id: string;
  event_type: string;
  event_date: string | null;
  deleted_at: string;
}

export interface TrashedDocument {
  id: string;
  case_id: string;
  person_id: string;
  document_type: string;
  title: string;
  deleted_at: string;
}

export interface TrashedClaimLine {
  id: string;
  case_id: string;
  root_person_id: string;
  status: string;
  deleted_at: string;
}

export interface CaseTrash {
  people: TrashedPerson[];
  life_events: TrashedLifeEvent[];
  documents: TrashedDocument[];
  claim_lines: TrashedClaimLine[];
}

export type TrashEntityType = 'people' | 'life-events' | 'documents' | 'claim-lines';

export function getCaseTrash(caseId: string): Promise<CaseTrash> {
  return get<CaseTrash>(`/cases/${caseId}/trash`);
}

export function getGlobalTrash(): Promise<GlobalTrash> {
  return get<GlobalTrash>('/trash');
}

export function restoreCase(caseId: string): Promise<void> {
  return post<void>(`/cases/${caseId}/restore`);
}

export function permanentDeleteCase(caseId: string): Promise<void> {
  return del(`/cases/${caseId}/permanent`);
}

export function restoreEntity(caseId: string, type: TrashEntityType, id: string): Promise<void> {
  return post<void>(`/cases/${caseId}/${type}/${id}/restore`);
}

export function permanentDeleteEntity(caseId: string, type: TrashEntityType, id: string): Promise<void> {
  return del(`/cases/${caseId}/${type}/${id}/permanent`);
}
