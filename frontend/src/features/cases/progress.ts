/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import type { Progress } from '../../api/cases';

/** A case's progress in a few words, for example "5 of 12 ready". */
export function progressSummary(p: Progress): { total: number; text: string } {
  const total = p.ready + p.in_progress + p.not_started;
  return { total, text: total === 0 ? 'No documents for submission yet' : `${p.ready} of ${total} ready` };
}
