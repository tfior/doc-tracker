/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

export const EVENT_TYPE_OPTIONS = [
  { value: 'birth', label: 'Birth' },
  { value: 'marriage', label: 'Marriage' },
  { value: 'death', label: 'Death' },
  { value: 'naturalization', label: 'Naturalization' },
  { value: 'immigration', label: 'Immigration' },
  { value: 'other', label: 'Other' },
];

export function eventTypeLabel(type: string) {
  return EVENT_TYPE_OPTIONS.find((o) => o.value === type)?.label ?? type;
}
