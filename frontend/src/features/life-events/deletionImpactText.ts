/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import type { LifeEventDeletionImpact } from '../../api/lifeevents';
import { count } from '../../lib/text';

/**
 * What goes with a life event, for a delete confirmation (domain model, rule
 * 78): to the trash with it, or, for an event already in the trash, permanently.
 */
export function lifeEventImpactText(impact: LifeEventDeletionImpact, permanent: boolean): string {
  const n = impact.documents;
  if (n === 0) {
    return permanent ? 'Nothing else is deleted with this event.' : 'It has no documents, so nothing else goes with it.';
  }
  return permanent
    ? `Also permanently deleted with this event: ${count(n, 'document')}.`
    : `Its ${count(n, 'document')} ${n === 1 ? 'goes' : 'go'} with it.`;
}
