/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

export const CASE_STATUS_OPTIONS = [
  { value: 'active', label: 'Active' },
  { value: 'archived', label: 'Archived' },
  { value: 'complete', label: 'Complete' },
];

export const CASE_STATUS_COLOR: Record<string, string> = {
  active: 'green',
  archived: 'gray',
  complete: 'blue',
};

export function caseStatusLabel(status: string) {
  return CASE_STATUS_OPTIONS.find((o) => o.value === status)?.label ?? status;
}
