/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

// A mirror of the server's date check, for feedback while typing. The server
// has the final say (architecture.md, Division of Responsibility).

const DATE_PATTERN = /^(\d{4})-(\d{2})-(\d{2})$/;

export const DATE_FORMAT_MESSAGE = 'Enter a real date in the form YYYY-MM-DD, for example 1890-03-15.';

/** Returns a message if value is neither empty nor a real date in the form YYYY-MM-DD. */
export function dateFormatError(value: string): string | null {
  const v = value.trim();
  if (v === '') return null;
  const m = DATE_PATTERN.exec(v);
  if (!m) return DATE_FORMAT_MESSAGE;
  const [year, month, day] = [Number(m[1]), Number(m[2]), Number(m[3])];
  // setUTCFullYear, unlike Date.UTC, does not read years below 100 as 19xx.
  const d = new Date(0);
  d.setUTCFullYear(year, month - 1, day);
  const real = d.getUTCFullYear() === year && d.getUTCMonth() === month - 1 && d.getUTCDate() === day;
  return real ? null : DATE_FORMAT_MESSAGE;
}
