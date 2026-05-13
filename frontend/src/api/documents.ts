import { get, post, patch, del, type ListResponse } from './client';

export interface PhaseStatus {
  id: string;
  label: string;
  phase: string;
  progress_bucket: string;
}

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
  official_copy_status: PhaseStatus;
  amendment_status: PhaseStatus;
  apostille_status: PhaseStatus;
  translation_status: PhaseStatus;
  created_at: string;
  updated_at: string;
}

export interface DocumentStatus {
  id: string;
  label: string;
  phase: string;
  is_system: boolean;
  progress_bucket: string;
}

export interface UpdateDocumentInput {
  title?: string;
  document_type?: string;
  issuing_authority?: string | null;
  issue_date?: string | null;
  recorded_date?: string | null;
  recorded_given_name?: string | null;
  recorded_surname?: string | null;
  recorded_birth_date?: string | null;
  recorded_birth_place?: string | null;
  notes?: string | null;
  is_verified?: boolean;
}

export function listDocuments(caseId: string): Promise<ListResponse<Document>> {
  return get<ListResponse<Document>>(`/cases/${caseId}/documents`);
}

export function listDocumentStatuses(): Promise<DocumentStatus[]> {
  return get<DocumentStatus[]>('/document-statuses');
}

export function createDocument(
  caseId: string,
  input: { person_id: string; document_type: string; title: string } & UpdateDocumentInput & { life_event_id?: string | null },
): Promise<Document> {
  return post<Document>(`/cases/${caseId}/documents`, input);
}

export function updateDocument(caseId: string, docId: string, input: UpdateDocumentInput): Promise<Document> {
  return patch<Document>(`/cases/${caseId}/documents/${docId}`, input);
}

export function deleteDocument(caseId: string, docId: string): Promise<void> {
  return del(`/cases/${caseId}/documents/${docId}`);
}

export function transitionStatus(
  caseId: string,
  docId: string,
  phase: string,
  statusId: string,
): Promise<Document> {
  return patch<Document>(`/cases/${caseId}/documents/${docId}/status`, { phase, status_id: statusId });
}

export function reassignDocument(
  caseId: string,
  docId: string,
  input: { person_id: string; life_event_id?: string | null },
): Promise<Document> {
  return patch<Document>(`/cases/${caseId}/documents/${docId}/parent`, input);
}
