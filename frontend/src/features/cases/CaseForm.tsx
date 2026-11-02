/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { Stack, Group, TextInput, Select, Button } from '@mantine/core';
import ErrorMessage from '../../components/ErrorMessage';
import type { CaseFormValues } from './caseFormValues';
import { fieldErrors } from '../../lib/errors';
import { CASE_STATUS_OPTIONS } from './options';

interface CaseFormProps {
  initial: CaseFormValues;
  isNew: boolean;
  saving: boolean;
  error: unknown;
  onSubmit: (values: CaseFormValues) => void;
  onCancel: () => void;
  onReload: () => Promise<void>;
}

export default function CaseForm({ initial, isNew, saving, error, onSubmit, onCancel, onReload }: CaseFormProps) {
  const [values, setValues] = useState(initial);
  const errors = fieldErrors(error);

  return (
    <Stack>
      <TextInput
        label="Title"
        placeholder="Case title"
        value={values.title}
        onChange={(e) => setValues({ ...values, title: e.currentTarget.value })}
        onKeyDown={(e) => e.key === 'Enter' && onSubmit(values)}
        error={errors.title}
        required
        data-autofocus
      />
      {!isNew && (
        <Select
          label="Status"
          data={CASE_STATUS_OPTIONS}
          value={values.status}
          onChange={(v) => setValues({ ...values, status: v ?? 'active' })}
          error={errors.status}
        />
      )}
      <ErrorMessage error={error} shownFields={['title', 'status']} onReload={onReload} />
      <Group justify="flex-end">
        <Button variant="default" onClick={onCancel} disabled={saving}>Cancel</Button>
        <Button onClick={() => onSubmit(values)} loading={saving}>
          {isNew ? 'Create' : 'Save'}
        </Button>
      </Group>
    </Stack>
  );
}
