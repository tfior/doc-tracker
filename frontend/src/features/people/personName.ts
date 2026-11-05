/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import type { Person } from '../../api/people';

export function fullName(p: Person) {
  return `${p.first_name} ${p.last_name}`;
}

/** The name of the person with this ID, or a dash if they are not in the list. */
export function personName(id: string, people: Person[]) {
  const p = people.find((p) => p.id === id);
  return p ? fullName(p) : '—';
}

export function personOptions(people: Person[]) {
  return people.map((p) => ({ value: p.id, label: fullName(p) }));
}

/**
 * The names of a life event's people: its person and, for a shared marriage,
 * the linked spouse. Anyone not in the list, such as a spouse in the trash
 * while the marriage stays with the other, is left out.
 */
export function eventPeople(le: { person_id: string; spouse_person_id: string | null }, people: Person[]) {
  const names = [le.person_id, le.spouse_person_id]
    .map((id) => people.find((p) => p.id === id))
    .filter((p): p is Person => p !== undefined)
    .map(fullName);
  return names.length > 0 ? names.join(' and ') : '—';
}
