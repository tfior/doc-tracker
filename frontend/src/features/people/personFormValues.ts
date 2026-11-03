/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import type { Person, PersonInput } from '../../api/people';
import { qualifiedDate, qualifiedDateParts, type QualifiedDateParts } from '../../lib/dates';

// Field values as typed. Blank text stands for "none", as does a blank date.
export interface PersonFormValues {
  first_name: string;
  last_name: string;
  birth_date: QualifiedDateParts;
  birth_place: string;
  death_date: QualifiedDateParts;
  death_place: string;
  notes: string;
  parent_ids: string[];
  child_ids: string[];
}

/** The form's starting values. A person's children are the people who list them as a parent. */
export function personFormValues(p: Person | null, people: Person[]): PersonFormValues {
  return {
    first_name: p?.first_name ?? '',
    last_name: p?.last_name ?? '',
    birth_date: qualifiedDateParts(p?.birth_date),
    birth_place: p?.birth_place ?? '',
    death_date: qualifiedDateParts(p?.death_date),
    death_place: p?.death_place ?? '',
    notes: p?.notes ?? '',
    parent_ids: p?.parent_ids ?? [],
    child_ids: p ? people.filter((other) => other.parent_ids.includes(p.id)).map((other) => other.id) : [],
  };
}

/** The request body for the values. ID lists are sorted so that order alone is not a change. */
export function personInput(v: PersonFormValues): PersonInput {
  return {
    first_name: v.first_name,
    last_name: v.last_name,
    birth_date: qualifiedDate(v.birth_date),
    birth_place: v.birth_place || null,
    death_date: qualifiedDate(v.death_date),
    death_place: v.death_place || null,
    notes: v.notes || null,
    parent_ids: [...v.parent_ids].sort(),
    child_ids: [...v.child_ids].sort(),
  };
}
