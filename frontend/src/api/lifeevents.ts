/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { get, post, patch, del, type Items } from './client';
import type { QualifiedDate } from './dates';

export interface LifeEvent {
  id: string;
  case_id: string;
  person_id: string;
  event_type: string;
  event_date: QualifiedDate | null;
  event_place: string | null;
  spouse_name: string | null;
  spouse_birth_date: QualifiedDate | null;
  spouse_birth_place: string | null;
  notes: string | null;
  has_documents: boolean;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface LifeEventInput {
  /** Changing the person moves the event's documents with it. */
  person_id?: string;
  event_type?: string;
  event_date?: QualifiedDate | null;
  event_place?: string | null;
  spouse_name?: string | null;
  spouse_birth_date?: QualifiedDate | null;
  spouse_birth_place?: string | null;
  notes?: string | null;
}

export interface LifeEventDeletionImpact {
  documents: number;
}

export function listLifeEvents(caseId: string): Promise<Items<LifeEvent>> {
  return get<Items<LifeEvent>>(`/cases/${caseId}/life-events`);
}

export function createLifeEvent(caseId: string, input: LifeEventInput): Promise<LifeEvent> {
  return post<LifeEvent>(`/cases/${caseId}/life-events`, input);
}

export function updateLifeEvent(caseId: string, eventId: string, version: number, input: LifeEventInput): Promise<LifeEvent> {
  return patch<LifeEvent>(`/cases/${caseId}/life-events/${eventId}`, { version, ...input });
}

export function deleteLifeEvent(caseId: string, eventId: string, version: number): Promise<void> {
  return del(`/cases/${caseId}/life-events/${eventId}`, { version });
}

export function getLifeEventDeletionImpact(caseId: string, eventId: string): Promise<LifeEventDeletionImpact> {
  return get<LifeEventDeletionImpact>(`/cases/${caseId}/life-events/${eventId}/deletion-impact`);
}
