/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { Group, Input, Select, TextInput } from '@mantine/core';
import { MONTHS, daysInMonth, isFourDigitYear, type DateParts } from '../lib/dates';

const MONTH_OPTIONS = MONTHS.map((name, i) => ({ value: String(i + 1), label: name }));

interface DatePartFieldsProps {
  value: DateParts;
  onChange: (value: DateParts) => void;
  /** Labels the fields for screen readers, such as "Start of range". */
  ariaLabel: string;
}

/**
 * Year, month and day fields for a date known only in part. Month and day are
 * optional, the day waits for a month, and only days that exist are offered.
 */
export function DatePartFields({ value, onChange, ariaLabel }: DatePartFieldsProps) {
  const [yearLeft, setYearLeft] = useState(false);
  const year = isFourDigitYear(value.year) ? Number(value.year) : null;
  const days = value.month ? daysInMonth(year, Number(value.month)) : 0;
  const dayOptions = Array.from({ length: days }, (_, i) => String(i + 1));

  // A day the new month or year does not have is cleared rather than kept.
  function update(next: DateParts) {
    const nextYear = isFourDigitYear(next.year) ? Number(next.year) : null;
    const fits = next.month && next.day && Number(next.day) <= daysInMonth(nextYear, Number(next.month));
    onChange({ ...next, day: fits ? next.day : '' });
  }

  let yearError: string | undefined;
  if (yearLeft && value.year !== '' && !isFourDigitYear(value.year)) yearError = 'Enter a four-digit year.';
  else if (value.year === '' && value.month !== '') yearError = 'Enter the year.';

  return (
    <Group gap="xs" align="flex-start" wrap="nowrap">
      <TextInput
        aria-label={`${ariaLabel}: year`}
        placeholder="Year"
        w={80}
        inputMode="numeric"
        maxLength={4}
        value={value.year}
        onChange={(e) => update({ ...value, year: e.currentTarget.value.replace(/\D/g, '') })}
        onBlur={() => setYearLeft(true)}
        error={yearError}
      />
      <Select
        aria-label={`${ariaLabel}: month`}
        placeholder="Month"
        w={130}
        data={MONTH_OPTIONS}
        value={value.month || null}
        onChange={(m) => update({ ...value, month: m ?? '', day: m ? value.day : '' })}
        clearable
      />
      <Select
        aria-label={`${ariaLabel}: day`}
        placeholder="Day"
        w={80}
        data={dayOptions}
        value={value.day || null}
        onChange={(d) => update({ ...value, day: d ?? '' })}
        disabled={!value.month}
        clearable
      />
    </Group>
  );
}

interface PartialDateInputProps {
  label: string;
  value: DateParts;
  onChange: (value: DateParts) => void;
  /** The server's message for this field, from the last save. */
  error?: string;
}

/** A date as a document gives it: a year, a year and month, or a full date. */
export default function PartialDateInput({ label, value, onChange, error }: PartialDateInputProps) {
  return (
    <Input.Wrapper label={label} error={error}>
      <DatePartFields value={value} onChange={onChange} ariaLabel={label} />
    </Input.Wrapper>
  );
}
