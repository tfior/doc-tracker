/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { Stack, Group, TextInput, MultiSelect, Button } from '@mantine/core';
import type { Person } from '../../api/people';
import DateField from '../../components/DateField';
import ErrorMessage from '../../components/ErrorMessage';
import type { PersonFormValues } from './personFormValues';
import { fieldErrors } from '../../lib/errors';
import { personOptions } from './personName';

const SHOWN_FIELDS = [
  'first_name', 'last_name', 'birth_date', 'birth_place', 'death_date', 'notes', 'parent_ids', 'child_ids',
];

interface PersonFormProps {
  initial: PersonFormValues;
  isNew: boolean;
  /** Everyone else in the case, who can be chosen as a parent or child. */
  others: Person[];
  saving: boolean;
  error: unknown;
  onSubmit: (values: PersonFormValues) => void;
  onCancel: () => void;
  onReload: () => Promise<void>;
}

export default function PersonForm({ initial, isNew, others, saving, error, onSubmit, onCancel, onReload }: PersonFormProps) {
  const [values, setValues] = useState(initial);
  const errors = fieldErrors(error);

  function set<K extends keyof PersonFormValues>(key: K, value: PersonFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  // Someone cannot be both a parent and a child of the same person.
  const options = personOptions(others);
  const parentOptions = options.filter((o) => !values.child_ids.includes(o.value));
  const childOptions = options.filter((o) => !values.parent_ids.includes(o.value));

  return (
    <Stack>
      <Group grow align="flex-start">
        <TextInput
          label="First Name"
          value={values.first_name}
          onChange={(e) => set('first_name', e.currentTarget.value)}
          error={errors.first_name}
          required
        />
        <TextInput
          label="Last Name"
          value={values.last_name}
          onChange={(e) => set('last_name', e.currentTarget.value)}
          error={errors.last_name}
          required
        />
      </Group>
      <Group grow align="flex-start">
        <DateField label="Birth Date" value={values.birth_date} onChange={(v) => set('birth_date', v)} error={errors.birth_date} />
        <TextInput
          label="Birth Place"
          value={values.birth_place}
          onChange={(e) => set('birth_place', e.currentTarget.value)}
          error={errors.birth_place}
        />
      </Group>
      <DateField label="Death Date" value={values.death_date} onChange={(v) => set('death_date', v)} error={errors.death_date} />
      <TextInput
        label="Notes"
        value={values.notes}
        onChange={(e) => set('notes', e.currentTarget.value)}
        error={errors.notes}
      />
      <MultiSelect
        label="Parents"
        description="Up to 2 people from this case"
        data={parentOptions}
        value={values.parent_ids}
        onChange={(v) => set('parent_ids', v)}
        maxValues={2}
        error={errors.parent_ids}
        searchable
        clearable
      />
      <MultiSelect
        label="Children"
        data={childOptions}
        value={values.child_ids}
        onChange={(v) => set('child_ids', v)}
        error={errors.child_ids}
        searchable
        clearable
      />
      <ErrorMessage error={error} shownFields={SHOWN_FIELDS} onReload={onReload} />
      <Group justify="flex-end">
        <Button variant="default" onClick={onCancel} disabled={saving}>Cancel</Button>
        <Button onClick={() => onSubmit(values)} loading={saving}>
          {isNew ? 'Add Person' : 'Save'}
        </Button>
      </Group>
    </Stack>
  );
}
