/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { TextInput } from '@mantine/core';
import { dateFormatError } from '../lib/dates';

interface DateFieldProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  /** The server's message for this field, from the last save. */
  error?: string;
}

/**
 * A date typed as YYYY-MM-DD. The format is checked as the user types, once a
 * full date's worth has been entered or the field is left, so a half-typed date
 * is not flagged. Used only for a person's birth and death dates, which move
 * into their life events in Milestone 4, taking this field with them.
 */
export default function DateField({ label, value, onChange, error }: DateFieldProps) {
  const [left, setLeft] = useState(false);
  const formatError = left || value.trim().length >= 10 ? dateFormatError(value) : null;

  return (
    <TextInput
      label={label}
      placeholder="YYYY-MM-DD"
      value={value}
      onChange={(e) => onChange(e.currentTarget.value)}
      onBlur={() => setLeft(true)}
      error={formatError ?? error}
    />
  );
}
