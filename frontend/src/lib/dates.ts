/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import type { DateQualifier, QualifiedDate } from '../api/dates';

// Mirrors of the server's date checks, for feedback while typing, and the
// formatting of dates for display. The server has the final say on what is
// valid (architecture.md, Division of Responsibility).

export const MONTHS = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December',
];

/** The number of days in a month. February has 29 when the year is not yet known. */
export function daysInMonth(year: number | null, month: number): number {
  if (month === 2 && year === null) return 29;
  // setUTCFullYear, unlike Date.UTC, does not read years below 100 as 19xx.
  const d = new Date(0);
  d.setUTCFullYear(year ?? 2000, month, 0);
  return d.getUTCDate();
}

/** A partial date as the three fields of the date input. Blank means not known. */
export interface DateParts {
  year: string;
  month: string; // "1" to "12"
  day: string;
}

export const NO_DATE: DateParts = { year: '', month: '', day: '' };

export function isFourDigitYear(year: string): boolean {
  return /^\d{4}$/.test(year) && year !== '0000';
}

export function dateParts(value: string | null | undefined): DateParts {
  if (!value) return NO_DATE;
  const [year = '', month = '', day = ''] = value.split('-');
  return { year, month: month ? String(Number(month)) : '', day: day ? String(Number(day)) : '' };
}

/** The partial date the parts make, or null when every part is blank. Only the
 * parts that are known are included, so 1890 and March stays "1890-03". A
 * month or day with no year is still sent, so the server refuses it with a
 * message instead of the date being cleared. */
export function partialDate(parts: DateParts): string | null {
  const year = parts.year.trim();
  if (year === '' && !parts.month && !parts.day) return null;
  if (!parts.month) return year;
  const month = parts.month.padStart(2, '0');
  if (!parts.day) return `${year}-${month}`;
  return `${year}-${month}-${parts.day.padStart(2, '0')}`;
}

/** Writes a partial date in words, day first: "4 March 1890", "March 1890", "1890". */
export function formatPartialDate(value: string): string {
  const { year, month, day } = dateParts(value);
  if (!month) return year;
  const monthName = MONTHS[Number(month) - 1];
  return day ? `${day} ${monthName} ${year}` : `${monthName} ${year}`;
}

export const QUALIFIER_OPTIONS: { value: DateQualifier; label: string }[] = [
  { value: 'exact', label: 'Exactly' },
  { value: 'about', label: 'About' },
  { value: 'before', label: 'Before' },
  { value: 'after', label: 'After' },
  { value: 'between', label: 'Between' },
];

/** Writes a qualified date in words: "4 March 1890", "about 1890", "between 1890 and 1892". */
export function formatQualifiedDate(date: QualifiedDate): string {
  const value = formatPartialDate(date.value);
  switch (date.qualifier) {
    case 'exact': return value;
    case 'between': return `between ${value} and ${date.end ? formatPartialDate(date.end) : '?'}`;
    default: return `${date.qualifier} ${value}`;
  }
}

/** A qualified date as the fields of the qualified date input. */
export interface QualifiedDateParts {
  qualifier: DateQualifier;
  start: DateParts;
  end: DateParts;
}

export function qualifiedDateParts(date: QualifiedDate | null | undefined): QualifiedDateParts {
  return {
    qualifier: date?.qualifier ?? 'exact',
    start: dateParts(date?.value),
    end: dateParts(date?.end),
  };
}

/** The qualified date the parts make, or null when the start is blank. An end
 * is sent only for a range, and only when some part of it is given; the server
 * reports one that is missing. */
export function qualifiedDate(parts: QualifiedDateParts): QualifiedDate | null {
  const value = partialDate(parts.start);
  if (value === null) return null;
  const end = parts.qualifier === 'between' ? partialDate(parts.end) : null;
  return end === null ? { value, qualifier: parts.qualifier } : { value, qualifier: parts.qualifier, end };
}
