/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import type { Document, DocumentInput, DocumentStatus, PhaseStatus } from '../../api/documents';

export const DOC_TYPE_OPTIONS = [
  { value: 'birth_certificate', label: 'Birth Certificate' },
  { value: 'marriage_certificate', label: 'Marriage Certificate' },
  { value: 'naturalization', label: 'Naturalization' },
  { value: 'death_certificate', label: 'Death Certificate' },
  { value: 'other', label: 'Other' },
];

export function docTypeLabel(type: string) {
  return DOC_TYPE_OPTIONS.find((o) => o.value === type)?.label ?? type;
}

export const PHASES = ['official_copy', 'amendment', 'apostille', 'translation'] as const;
export type Phase = typeof PHASES[number];

export const PHASE_LABELS: Record<Phase, string> = {
  official_copy: 'Official Copy',
  amendment: 'Amendment',
  apostille: 'Apostille',
  translation: 'Translation',
};

/** The request field that sets a phase's status. */
export function phaseStatusField(phase: Phase) {
  return `${phase}_status_id` as const satisfies keyof DocumentInput;
}

export function phaseStatus(doc: Document, phase: Phase): PhaseStatus {
  return doc[`${phase}_status`];
}

/** The statuses a phase can take, as select options. */
export function phaseStatusOptions(statuses: DocumentStatus[], phase: Phase) {
  return statuses
    .filter((s) => s.phase === phase || s.phase === 'any')
    .map((s) => ({ value: s.id, label: s.label }));
}
