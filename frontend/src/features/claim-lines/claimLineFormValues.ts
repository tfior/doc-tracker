/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import type { ClaimLine } from '../../api/claimlines';

export interface ClaimLineFormValues {
  root_person_id: string;
  status: string;
  notes: string;
}

export function claimLineFormValues(line: ClaimLine | null): ClaimLineFormValues {
  return {
    root_person_id: line?.root_person_id ?? '',
    status: line?.status ?? 'not_yet_researched',
    notes: line?.notes ?? '',
  };
}
