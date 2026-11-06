/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

// Wording for the readiness the server reports. The rules are the server's;
// this file only names what it returns.

import type { Document, Readiness, Stage } from '../../api/documents';
import { PHASES, PHASE_LABELS, type Phase } from './options';

export const READINESS_LABELS: Record<Readiness['status'], string> = {
  ready: 'Ready',
  in_progress: 'In Progress',
  not_started: 'Not Started',
};

export const READINESS_COLOR: Record<Readiness['status'], string> = {
  ready: 'green',
  in_progress: 'blue',
  not_started: 'gray',
};

/** The stages in the order a document waits on them. */
export const STAGES: Stage[] = [...PHASES, 'details_check'];

export const STAGE_LABELS: Record<Stage, string> = {
  ...PHASE_LABELS,
  details_check: 'Details Check',
};

/** A reason code as a sentence fragment, for example "Apostille / Legalization not complete". */
export function reasonLabel(code: string): string {
  if (code === 'details_not_checked') return 'Details not checked against scan';
  const phase = code.replace(/_incomplete$/, '');
  if (phase in PHASE_LABELS) return `${PHASE_LABELS[phase as Phase]} not complete`;
  return code;
}

/** Why a document has no readiness: it is outside the packet. */
export function outsidePacketLabel(doc: Document): string {
  return doc.needed ? 'Not for submission' : 'Not needed';
}

export function daysInStatus(doc: Document, phase: Phase): number {
  return doc[`${phase}_days_in_status`];
}
