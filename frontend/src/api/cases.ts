/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { get, post, patch, del, type Items } from './client';

export interface Case {
  id: string;
  title: string;
  status: string;
  primary_root_person_id: string | null;
  version: number;
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

interface CasePage extends Items<Case> {
  page: number;
  per_page: number;
}

// The largest page the server allows.
const PER_PAGE = 200;

/** Fetches every case, one page after another. */
export async function listCases(): Promise<Items<Case>> {
  const items: Case[] = [];
  for (let page = 1; ; page++) {
    const res = await get<CasePage>('/cases', { page, per_page: PER_PAGE });
    items.push(...res.items);
    if (res.items.length === 0 || items.length >= res.total) {
      return { items, total: res.total };
    }
  }
}

export function getCase(caseId: string): Promise<CaseDetail> {
  return get<CaseDetail>(`/cases/${caseId}`);
}

export function createCase(title: string): Promise<Case> {
  return post<Case>('/cases', { title });
}

export function updateCase(caseId: string, version: number, input: { title?: string; status?: string }): Promise<Case> {
  return patch<Case>(`/cases/${caseId}`, { version, ...input });
}

export function deleteCase(caseId: string, version: number): Promise<void> {
  return del(`/cases/${caseId}`, { version });
}
