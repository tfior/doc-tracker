/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import type { LifeEvent, LifeEventInput } from '../../api/lifeevents';

// Field values as typed. Blank text stands for "none".
export interface LifeEventFormValues {
  person_id: string;
  event_type: string;
  event_date: string;
  event_place: string;
  spouse_name: string;
  spouse_birth_date: string;
  spouse_birth_place: string;
  notes: string;
}

export function lifeEventFormValues(le: LifeEvent | null, personId?: string): LifeEventFormValues {
  return {
    person_id: le?.person_id ?? personId ?? '',
    event_type: le?.event_type ?? '',
    event_date: le?.event_date ?? '',
    event_place: le?.event_place ?? '',
    spouse_name: le?.spouse_name ?? '',
    spouse_birth_date: le?.spouse_birth_date ?? '',
    spouse_birth_place: le?.spouse_birth_place ?? '',
    notes: le?.notes ?? '',
  };
}

/** The request body for the values. Spouse details are kept only for a marriage. */
export function lifeEventInput(v: LifeEventFormValues): LifeEventInput {
  const isMarriage = v.event_type === 'marriage';
  return {
    person_id: v.person_id,
    event_type: v.event_type,
    event_date: v.event_date || null,
    event_place: v.event_place || null,
    spouse_name: isMarriage ? (v.spouse_name || null) : null,
    spouse_birth_date: isMarriage ? (v.spouse_birth_date || null) : null,
    spouse_birth_place: isMarriage ? (v.spouse_birth_place || null) : null,
    notes: v.notes || null,
  };
}
