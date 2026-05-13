import { get, post, patch, del, type ListResponse } from './client';

export interface ClaimLine {
  id: string;
  case_id: string;
  root_person_id: string;
  status: string;
  notes: string | null;
  created_at: string;
  updated_at: string;
}

export function listClaimLines(caseId: string): Promise<ListResponse<ClaimLine>> {
  return get<ListResponse<ClaimLine>>(`/cases/${caseId}/claim-lines`);
}

export function createClaimLine(
  caseId: string,
  input: { root_person_id: string; status: string; notes?: string | null },
): Promise<ClaimLine> {
  return post<ClaimLine>(`/cases/${caseId}/claim-lines`, input);
}

export function updateClaimLine(
  caseId: string,
  lineId: string,
  input: { status?: string; notes?: string | null },
): Promise<ClaimLine> {
  return patch<ClaimLine>(`/cases/${caseId}/claim-lines/${lineId}`, input);
}

export function deleteClaimLine(caseId: string, lineId: string): Promise<void> {
  return del(`/cases/${caseId}/claim-lines/${lineId}`);
}
