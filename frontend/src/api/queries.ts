/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

// Every query the app makes, so that each list has one cache key wherever it is
// read, refreshed, or reloaded after a version conflict.

import { queryOptions, type QueryClient } from '@tanstack/react-query';
import { listCases, getCase } from './cases';
import { listPeople } from './people';
import { listLifeEvents } from './lifeevents';
import { listDocuments, listDocumentStatuses } from './documents';
import { listClaimLines } from './claimlines';
import { getCaseTrash, getGlobalTrash } from './trash';

export const casesQuery = () =>
  queryOptions({ queryKey: ['cases'], queryFn: listCases });

export const caseQuery = (caseId: string) =>
  queryOptions({ queryKey: ['cases', caseId], queryFn: () => getCase(caseId) });

export const peopleQuery = (caseId: string) =>
  queryOptions({ queryKey: ['people', caseId], queryFn: () => listPeople(caseId) });

export const lifeEventsQuery = (caseId: string) =>
  queryOptions({ queryKey: ['life-events', caseId], queryFn: () => listLifeEvents(caseId) });

export const documentsQuery = (caseId: string) =>
  queryOptions({ queryKey: ['documents', caseId], queryFn: () => listDocuments(caseId) });

export const claimLinesQuery = (caseId: string) =>
  queryOptions({ queryKey: ['claim-lines', caseId], queryFn: () => listClaimLines(caseId) });

export const documentStatusesQuery = () =>
  queryOptions({ queryKey: ['document-statuses'], queryFn: listDocumentStatuses, staleTime: 5 * 60 * 1000 });

export const caseTrashQuery = (caseId: string) =>
  queryOptions({ queryKey: ['trash', caseId], queryFn: () => getCaseTrash(caseId) });

export const globalTrashQuery = () =>
  queryOptions({ queryKey: ['global-trash'], queryFn: getGlobalTrash });

/**
 * Marks everything shown for a case as out of date after a write. A change to
 * one record can change others (moving a life event moves its documents;
 * setting a person's children changes their parents), so the whole case is
 * refreshed rather than guessing which lists a write touched.
 */
export function invalidateCase(queryClient: QueryClient, caseId: string) {
  for (const key of [['cases'], ['people', caseId], ['life-events', caseId], ['documents', caseId], ['claim-lines', caseId], ['trash', caseId], ['global-trash']]) {
    queryClient.invalidateQueries({ queryKey: key });
  }
}
