/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

// Dates that may be known only in part (api-conventions.md, Dates). A partial
// date is a string: "1890", "1890-03" or "1890-03-15".

export type DateQualifier = 'exact' | 'about' | 'before' | 'after' | 'between';

/** A partial date with how well it is known; `end` is set only for "between". */
export interface QualifiedDate {
  value: string;
  qualifier: DateQualifier;
  end?: string;
}
