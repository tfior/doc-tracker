/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { Stack, Group, Select, Textarea, Text, Button } from '@mantine/core';
import type { Person } from '../../api/people';
import ErrorMessage from '../../components/ErrorMessage';
import type { ClaimLineFormValues } from './claimLineFormValues';
import { fieldErrors } from '../../lib/errors';
import { personName, personOptions } from '../people/personName';
import { CLAIM_STATUS_OPTIONS } from './options';

const SHOWN_FIELDS = ['root_person_id', 'status', 'notes'];

interface ClaimLineFormProps {
  initial: ClaimLineFormValues;
  isNew: boolean;
  people: Person[];
  saving: boolean;
  error: unknown;
  onSubmit: (values: ClaimLineFormValues) => void;
  onCancel: () => void;
  onReload: () => Promise<void>;
}

export default function ClaimLineForm({ initial, isNew, people, saving, error, onSubmit, onCancel, onReload }: ClaimLineFormProps) {
  const [values, setValues] = useState(initial);
  const errors = fieldErrors(error);

  return (
    <Stack>
      {isNew ? (
        <Select
          label="Root Person"
          description="The lineage-relevant ancestor this claim line traces from"
          data={personOptions(people)}
          value={values.root_person_id}
          onChange={(v) => setValues({ ...values, root_person_id: v ?? '' })}
          error={errors.root_person_id}
          searchable
          required
        />
      ) : (
        <Text size="sm" c="dimmed">
          Root person: <strong>{personName(values.root_person_id, people)}</strong>
        </Text>
      )}
      <Select
        label="Status"
        data={CLAIM_STATUS_OPTIONS}
        value={values.status}
        onChange={(v) => setValues({ ...values, status: v ?? 'not_yet_researched' })}
        error={errors.status}
      />
      <Textarea
        label="Notes"
        value={values.notes}
        onChange={(e) => setValues({ ...values, notes: e.currentTarget.value })}
        error={errors.notes}
        autosize
        minRows={2}
      />
      <ErrorMessage error={error} shownFields={SHOWN_FIELDS} onReload={onReload} />
      <Group justify="flex-end">
        <Button variant="default" onClick={onCancel} disabled={saving}>Cancel</Button>
        <Button onClick={() => onSubmit(values)} loading={saving}>
          {isNew ? 'Add' : 'Save'}
        </Button>
      </Group>
    </Stack>
  );
}
