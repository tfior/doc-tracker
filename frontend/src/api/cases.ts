import { get, post, patch, del } from './client';

export interface Case {
  id: string;
  title: string;
  status: string;
  primary_root_person_id: string | null;
  created_at: string;
  updated_at: string;
}

export interface ClaimLineSummary {
  total: number;
  not_yet_researched: number;
  researching: number;
  paused: number;
  ineligible: number;
  eligible: number;
}

export interface CaseDetail extends Case {
  claim_line_summary: ClaimLineSummary;
}

export interface ListResponse<T> {
  items: T[];
  total: number;
  page: number;
  per_page: number;
}

export function listCases(): Promise<ListResponse<Case>> {
  return get<ListResponse<Case>>('/cases');
}

export function getCase(caseId: string): Promise<CaseDetail> {
  return get<CaseDetail>(`/cases/${caseId}`);
}

export function createCase(title: string): Promise<Case> {
  return post<Case>('/cases', { title });
}

export function updateCase(caseId: string, input: { title?: string; status?: string }): Promise<Case> {
  return patch<Case>(`/cases/${caseId}`, input);
}

export function deleteCase(caseId: string): Promise<void> {
  return del(`/cases/${caseId}`);
}
