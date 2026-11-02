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
