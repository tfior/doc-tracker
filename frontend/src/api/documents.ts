/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { get, post, patch, del, type Items } from './client';

export interface PhaseStatus {
  id: string;
  label: string;
  phase: string;
  progress_bucket: string;
}

/** A user named in a response. */
export interface UserRef {
  id: string;
  name: string;
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
  registration_date: string | null;
  recorded_given_name: string | null;
  recorded_surname: string | null;
  recorded_birth_date: string | null;
  recorded_birth_place: string | null;
  details_checked: boolean;
  details_checked_at: string | null;
  details_checked_by: UserRef | null;
  for_submission: boolean;
  notes: string | null;
  official_copy_status: PhaseStatus;
  official_copy_status_since: string;
  official_copy_reference: string | null;
  amendment_status: PhaseStatus;
  amendment_status_since: string;
  amendment_reference: string | null;
  apostille_status: PhaseStatus;
  apostille_status_since: string;
  apostille_reference: string | null;
  translation_status: PhaseStatus;
  translation_status_since: string;
  translation_reference: string | null;
  version: number;
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

export interface DocumentInput {
  /** Moving a document to another person without naming an event ungroups it. */
  person_id?: string;
  life_event_id?: string | null;
  title?: string;
  document_type?: string;
  issuing_authority?: string | null;
  issue_date?: string | null;
  registration_date?: string | null;
  recorded_given_name?: string | null;
  recorded_surname?: string | null;
  recorded_birth_date?: string | null;
  recorded_birth_place?: string | null;
  notes?: string | null;
  /** Sending true with a changed recorded detail records a new check; changing one without it clears the check. */
  details_checked?: boolean;
  for_submission?: boolean;
  official_copy_status_id?: string;
  amendment_status_id?: string;
  apostille_status_id?: string;
  translation_status_id?: string;
  /** Sent only to correct the date; a status change otherwise dates itself. */
  official_copy_status_since?: string | null;
  amendment_status_since?: string | null;
  apostille_status_since?: string | null;
  translation_status_since?: string | null;
  official_copy_reference?: string | null;
  amendment_reference?: string | null;
  apostille_reference?: string | null;
  translation_reference?: string | null;
}

export function listDocuments(caseId: string): Promise<Items<Document>> {
  return get<Items<Document>>(`/cases/${caseId}/documents`);
}

export function listDocumentStatuses(): Promise<DocumentStatus[]> {
  return get<DocumentStatus[]>('/document-statuses');
}

export function createDocument(caseId: string, input: DocumentInput): Promise<Document> {
  return post<Document>(`/cases/${caseId}/documents`, input);
}

export function updateDocument(caseId: string, docId: string, version: number, input: DocumentInput): Promise<Document> {
  return patch<Document>(`/cases/${caseId}/documents/${docId}`, { version, ...input });
}

export function deleteDocument(caseId: string, docId: string, version: number): Promise<void> {
  return del(`/cases/${caseId}/documents/${docId}`, { version });
}
