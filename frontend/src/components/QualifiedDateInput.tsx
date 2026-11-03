/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { Group, Input, Select, Stack, Text } from '@mantine/core';
import type { DateQualifier } from '../api/dates';
import { QUALIFIER_OPTIONS, type QualifiedDateParts } from '../lib/dates';
import { DatePartFields } from './PartialDateInput';

interface QualifiedDateInputProps {
  label: string;
  value: QualifiedDateParts;
  onChange: (value: QualifiedDateParts) => void;
  /** The server's messages for this field and its parts, by field name. */
  errors: Record<string, string>;
  /** The field's name in the API, such as "event_date", for finding its errors. */
  field: string;
}

/**
 * A date as it is believed to be: a partial date and how well it is known,
 * and for "between", a second date ending the range.
 */
export default function QualifiedDateInput({ label, value, onChange, errors, field }: QualifiedDateInputProps) {
  const error = [errors[field], errors[`${field}.value`], errors[`${field}.qualifier`], errors[`${field}.end`]]
    .filter(Boolean).join(' ') || undefined;

  return (
    <Input.Wrapper label={label} error={error}>
      <Stack gap="xs">
        <Group gap="xs" align="flex-start" wrap="nowrap">
          <Select
            aria-label={`${label}: how well it is known`}
            w={110}
            data={QUALIFIER_OPTIONS}
            value={value.qualifier}
            onChange={(q) => onChange({ ...value, qualifier: (q ?? 'exact') as DateQualifier })}
            allowDeselect={false}
          />
          <DatePartFields
            value={value.start}
            onChange={(start) => onChange({ ...value, start })}
            ariaLabel={value.qualifier === 'between' ? `${label}, start of range` : label}
          />
        </Group>
        {value.qualifier === 'between' && (
          <Group gap="xs" align="center" wrap="nowrap">
            <Text size="sm" w={110} ta="right">and</Text>
            <DatePartFields
              value={value.end}
              onChange={(end) => onChange({ ...value, end })}
              ariaLabel={`${label}, end of range`}
            />
          </Group>
        )}
      </Stack>
    </Input.Wrapper>
  );
}
