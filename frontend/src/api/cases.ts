import { get, type ListResponse } from './client';

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
  active: number;
  suspended: number;
  eliminated: number;
  confirmed: number;
}

export interface DocumentProgress {
  not_started: number;
  in_progress: number;
  complete: number;
}

export interface CaseDetail extends Case {
  claim_line_summary: ClaimLineSummary;
  document_progress: DocumentProgress;
}

export function listCases(): Promise<ListResponse<Case>> {
  return get<ListResponse<Case>>('/cases');
}

export function getCase(caseId: string): Promise<CaseDetail> {
  return get<CaseDetail>(`/cases/${caseId}`);
}
