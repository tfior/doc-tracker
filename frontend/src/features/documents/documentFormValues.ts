/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import type { Document, DocumentInput } from '../../api/documents';
import { PHASES, phaseReferenceField, phaseSinceField, phaseStatus, phaseStatusField, type Phase } from './options';
import { dateParts, localDay, partialDate, startOfLocalDay, type DateParts } from '../../lib/dates';

/** One phase as typed: its status, the local day it entered it, and its reference. */
export interface PhaseValues {
  status_id: string;
  since: string;
  /** True while the since date differs from the saved one; only then is it sent. */
  since_edited: boolean;
  reference: string;
}

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
  details_checked: boolean;
  for_submission: boolean;
  phases: Record<Phase, PhaseValues>;
}

/** The fields the details check attests to. Editing one clears the check in the form. */
export const RECORDED_FIELDS = [
  'recorded_given_name', 'recorded_surname', 'recorded_birth_date', 'recorded_birth_place',
] as const satisfies readonly (keyof DocumentFormValues)[];

function phaseValues(doc: Document | null): Record<Phase, PhaseValues> {
  return Object.fromEntries(PHASES.map((phase) => [phase, {
    status_id: doc ? phaseStatus(doc, phase).id : '',
    since: doc ? localDay(doc[phaseSinceField(phase)]) : '',
    since_edited: false,
    reference: doc?.[phaseReferenceField(phase)] ?? '',
  }])) as Record<Phase, PhaseValues>;
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
    details_checked: doc?.details_checked ?? false,
    for_submission: doc?.for_submission ?? true,
    phases: phaseValues(doc),
  };
}

/**
 * The request body for the values. A new document takes its phases from the
 * server. A since date is sent only when the user corrected it, so a status
 * change the user did not date is dated by the server.
 */
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
    details_checked: v.details_checked,
    for_submission: v.for_submission,
  };
  if (!isNew) {
    for (const phase of PHASES) {
      const p = v.phases[phase];
      input[phaseStatusField(phase)] = p.status_id;
      input[phaseReferenceField(phase)] = p.reference.trim() || null;
      if (p.since_edited) {
        // A cleared date is sent as null, which the server refuses with a field error.
        input[phaseSinceField(phase)] = p.since ? startOfLocalDay(p.since) : null;
      }
    }
  }
  return input;
}
