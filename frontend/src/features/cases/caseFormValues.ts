/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import type { Case } from '../../api/cases';

export interface CaseFormValues {
  title: string;
  status: string;
}

export function caseFormValues(c: Case | null): CaseFormValues {
  return { title: c?.title ?? '', status: c?.status ?? 'active' };
}
