/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import type { PersonDeletionImpact } from '../../api/people';
import { count, joinList } from '../../lib/text';

/**
 * What goes with a person, for a delete confirmation (domain model, rule 78):
 * to the trash with them, or, for a person already in the trash, permanently.
 */
export function personImpactText(impact: PersonDeletionImpact, permanent: boolean): string {
  const parts = [
    impact.life_events > 0 && count(impact.life_events, 'life event'),
    impact.documents > 0 && count(impact.documents, 'document'),
    impact.claim_lines > 0 && count(impact.claim_lines, 'claim line'),
  ].filter((p): p is string => !!p);

  if (parts.length === 0) {
    return permanent ? 'Nothing else is deleted with this person.' : 'Nothing else goes with this person.';
  }
  return permanent
    ? `Also permanently deleted with this person: ${joinList(parts)}.`
    : `Going with this person: ${joinList(parts)}.`;
}
