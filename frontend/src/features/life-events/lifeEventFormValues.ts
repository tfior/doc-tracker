/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import type { LifeEvent, LifeEventInput } from '../../api/lifeevents';
import { qualifiedDate, qualifiedDateParts, type QualifiedDateParts } from '../../lib/dates';

// Field values as typed. Blank text stands for "none", as does a date with no year.
export interface LifeEventFormValues {
  person_id: string;
  event_type: string;
  event_date: QualifiedDateParts;
  event_place: string;
  /** Blank when the spouse is not a person in the case. */
  spouse_person_id: string;
  spouse_name: string;
  spouse_birth_date: QualifiedDateParts;
  spouse_birth_place: string;
  divorced: boolean;
  divorce_date: QualifiedDateParts;
  divorce_place: string;
  notes: string;
}

export function lifeEventFormValues(le: LifeEvent | null, personId?: string): LifeEventFormValues {
  return {
    person_id: le?.person_id ?? personId ?? '',
    event_type: le?.event_type ?? '',
    event_date: qualifiedDateParts(le?.event_date),
    event_place: le?.event_place ?? '',
    spouse_person_id: le?.spouse_person_id ?? '',
    spouse_name: le?.spouse_name ?? '',
    spouse_birth_date: qualifiedDateParts(le?.spouse_birth_date),
    spouse_birth_place: le?.spouse_birth_place ?? '',
    divorced: Boolean(le?.divorce_date || le?.divorce_place),
    divorce_date: qualifiedDateParts(le?.divorce_date),
    divorce_place: le?.divorce_place ?? '',
    notes: le?.notes ?? '',
  };
}

/**
 * The request body for the values. Spouse and divorce details are kept only for
 * a marriage, the spouse's own details only when they are not a linked person,
 * and divorce details only when the marriage ended in divorce.
 */
export function lifeEventInput(v: LifeEventFormValues): LifeEventInput {
  const isMarriage = v.event_type === 'marriage';
  const flat = isMarriage && !v.spouse_person_id;
  const divorced = isMarriage && v.divorced;
  return {
    person_id: v.person_id,
    event_type: v.event_type,
    event_date: qualifiedDate(v.event_date),
    event_place: v.event_place || null,
    spouse_person_id: isMarriage ? (v.spouse_person_id || null) : null,
    spouse_name: flat ? (v.spouse_name || null) : null,
    spouse_birth_date: flat ? qualifiedDate(v.spouse_birth_date) : null,
    spouse_birth_place: flat ? (v.spouse_birth_place || null) : null,
    divorce_date: divorced ? qualifiedDate(v.divorce_date) : null,
    divorce_place: divorced ? (v.divorce_place || null) : null,
    notes: v.notes || null,
  };
}
