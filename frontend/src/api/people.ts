/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { get, post, patch, del, type Items } from './client';
import type { QualifiedDate } from './dates';

export interface Person {
  id: string;
  case_id: string;
  first_name: string;
  last_name: string;
  /** Birth and death are read from the person's birth and death life events. */
  birth_date: QualifiedDate | null;
  birth_place: string | null;
  death_date: QualifiedDate | null;
  death_place: string | null;
  notes: string | null;
  parent_ids: string[];
  version: number;
  created_at: string;
  updated_at: string;
}

export interface PersonInput {
  first_name?: string;
  last_name?: string;
  /** Birth and death are saved to the person's birth and death life events. */
  birth_date?: QualifiedDate | null;
  birth_place?: string | null;
  death_date?: QualifiedDate | null;
  death_place?: string | null;
  notes?: string | null;
  /** The full list of parents: what is sent replaces the current links. */
  parent_ids?: string[];
  /** The full list of children: what is sent replaces the current links. */
  child_ids?: string[];
}

export interface PersonDeletionImpact {
  life_events: number;
  documents: number;
  claim_lines: number;
}

export function listPeople(caseId: string): Promise<Items<Person>> {
  return get<Items<Person>>(`/cases/${caseId}/people`);
}

export function createPerson(caseId: string, input: PersonInput): Promise<Person> {
  return post<Person>(`/cases/${caseId}/people`, input);
}

export function updatePerson(caseId: string, personId: string, version: number, input: PersonInput): Promise<Person> {
  return patch<Person>(`/cases/${caseId}/people/${personId}`, { version, ...input });
}

export function deletePerson(caseId: string, personId: string, version: number): Promise<void> {
  return del(`/cases/${caseId}/people/${personId}`, { version });
}

export function getPersonDeletionImpact(caseId: string, personId: string): Promise<PersonDeletionImpact> {
  return get<PersonDeletionImpact>(`/cases/${caseId}/people/${personId}/deletion-impact`);
}
