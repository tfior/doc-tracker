import { get, type ListResponse } from './client';

export interface Document {
  id: string;
  case_id: string;
  person_id: string;
  life_event_id: string | null;
  document_type: string;
  title: string;
  issuing_authority: string | null;
  issue_date: string | null;
  recorded_date: string | null;
  recorded_given_name: string | null;
  recorded_surname: string | null;
  recorded_birth_date: string | null;
  recorded_birth_place: string | null;
  is_verified: boolean;
  verified_at: string | null;
  notes: string | null;
  status: string;
  status_key: string | null;
  progress_bucket: string;
  created_at: string;
  updated_at: string;
}

export function listDocuments(caseId: string): Promise<ListResponse<Document>> {
  return get<ListResponse<Document>>(`/cases/${caseId}/documents`);
}
