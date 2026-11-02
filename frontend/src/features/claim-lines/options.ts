/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

export const CLAIM_STATUS_OPTIONS = [
  { value: 'not_yet_researched', label: 'Not Yet Researched' },
  { value: 'researching',        label: 'Researching' },
  { value: 'paused',             label: 'Paused' },
  { value: 'ineligible',         label: 'Ineligible' },
  { value: 'eligible',           label: 'Eligible' },
];

export const CLAIM_STATUS_COLOR: Record<string, string> = {
  not_yet_researched: 'gray',
  researching:        'blue',
  paused:             'orange',
  ineligible:         'red',
  eligible:           'green',
};

export function claimStatusLabel(status: string) {
  return CLAIM_STATUS_OPTIONS.find((o) => o.value === status)?.label ?? status;
}
