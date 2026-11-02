/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { ApiError } from '../api/client';

/** The message to show for a failed request: the server's own, when it sent one. */
export function errorMessage(err: unknown): string {
  if (err instanceof ApiError) return err.message;
  return 'Something unexpected went wrong. Please try again.';
}

/** Field messages by field name, for showing beside each field. */
export function fieldErrors(err: unknown): Record<string, string> {
  const out: Record<string, string> = {};
  if (!(err instanceof ApiError)) return out;
  for (const f of err.fields) {
    out[f.field] = out[f.field] ? `${out[f.field]} ${f.message}` : f.message;
  }
  return out;
}

/** True when a save was refused because the record changed since it was read. */
export function isVersionConflict(err: unknown): boolean {
  return err instanceof ApiError && err.code === 'version_conflict';
}
