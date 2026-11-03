/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import type { Document, DocumentInput } from '../../api/documents';
import { PHASES, phaseStatus, phaseStatusField } from './options';
import { dateParts, partialDate, type DateParts } from '../../lib/dates';

// Field values as typed. Blank text stands for "none", as does a date with no year.
export interface DocumentFormValues {
  person_id: string;
  life_event_id: string;
  document_type: string;
  title: string;
  issuing_authority: string;
  issue_date: DateParts;
  registration_date: DateParts;
  recorded_given_name: string;
  recorded_surname: string;
  recorded_birth_date: DateParts;
  recorded_birth_place: string;
  notes: string;
  is_verified: boolean;
  official_copy_status_id: string;
  amendment_status_id: string;
  apostille_status_id: string;
  translation_status_id: string;
}

export function documentFormValues(
  doc: Document | null,
  defaults: { personId?: string; lifeEventId?: string } = {},
): DocumentFormValues {
  return {
    person_id: doc?.person_id ?? defaults.personId ?? '',
    life_event_id: doc ? (doc.life_event_id ?? '') : (defaults.lifeEventId ?? ''),
    document_type: doc?.document_type ?? '',
    title: doc?.title ?? '',
    issuing_authority: doc?.issuing_authority ?? '',
    issue_date: dateParts(doc?.issue_date),
    registration_date: dateParts(doc?.registration_date),
    recorded_given_name: doc?.recorded_given_name ?? '',
    recorded_surname: doc?.recorded_surname ?? '',
    recorded_birth_date: dateParts(doc?.recorded_birth_date),
    recorded_birth_place: doc?.recorded_birth_place ?? '',
    notes: doc?.notes ?? '',
    is_verified: doc?.is_verified ?? false,
    official_copy_status_id: doc ? phaseStatus(doc, 'official_copy').id : '',
    amendment_status_id: doc ? phaseStatus(doc, 'amendment').id : '',
    apostille_status_id: doc ? phaseStatus(doc, 'apostille').id : '',
    translation_status_id: doc ? phaseStatus(doc, 'translation').id : '',
  };
}

/** The request body for the values. A new document takes its phase statuses from the server. */
export function documentInput(v: DocumentFormValues, isNew: boolean): DocumentInput {
  const input: DocumentInput = {
    person_id: v.person_id,
    life_event_id: v.life_event_id || null,
    document_type: v.document_type,
    title: v.title,
    issuing_authority: v.issuing_authority || null,
    issue_date: partialDate(v.issue_date),
    registration_date: partialDate(v.registration_date),
    recorded_given_name: v.recorded_given_name || null,
    recorded_surname: v.recorded_surname || null,
    recorded_birth_date: partialDate(v.recorded_birth_date),
    recorded_birth_place: v.recorded_birth_place || null,
    notes: v.notes || null,
    is_verified: v.is_verified,
  };
  if (!isNew) {
    for (const phase of PHASES) input[phaseStatusField(phase)] = v[phaseStatusField(phase)];
  }
  return input;
}
