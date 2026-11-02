/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { Stack, Group, TextInput, Select, Button } from '@mantine/core';
import type { Person } from '../../api/people';
import DateField from '../../components/DateField';
import ErrorMessage from '../../components/ErrorMessage';
import type { LifeEventFormValues } from './lifeEventFormValues';
import { fieldErrors } from '../../lib/errors';
import { personOptions } from '../people/personName';
import { EVENT_TYPE_OPTIONS } from './options';

const SHOWN_FIELDS = [
  'person_id', 'event_type', 'event_date', 'event_place',
  'spouse_name', 'spouse_birth_date', 'spouse_birth_place', 'notes',
];

interface LifeEventFormProps {
  initial: LifeEventFormValues;
  isNew: boolean;
  people: Person[];
  /** Hides the person field; the event stays with its person. */
  hidePerson: boolean;
  saving: boolean;
  error: unknown;
  onSubmit: (values: LifeEventFormValues) => void;
  onCancel: () => void;
  onReload: () => Promise<void>;
}

export default function LifeEventForm({
  initial, isNew, people, hidePerson, saving, error, onSubmit, onCancel, onReload,
}: LifeEventFormProps) {
  const [values, setValues] = useState(initial);
  const errors = fieldErrors(error);

  function set<K extends keyof LifeEventFormValues>(key: K, value: LifeEventFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  return (
    <Stack>
      {!hidePerson && (
        <Select
          label="Person"
          description={isNew ? undefined : "Changing the person moves this event's documents with it."}
          data={personOptions(people)}
          value={values.person_id}
          onChange={(v) => set('person_id', v ?? '')}
          error={errors.person_id}
          searchable
          required
        />
      )}
      <Select
        label="Event Type"
        data={EVENT_TYPE_OPTIONS}
        value={values.event_type}
        onChange={(v) => set('event_type', v ?? '')}
        error={errors.event_type}
        required
      />
      <Group grow align="flex-start">
        <DateField label="Date" value={values.event_date} onChange={(v) => set('event_date', v)} error={errors.event_date} />
        <TextInput
          label="Place"
          value={values.event_place}
          onChange={(e) => set('event_place', e.currentTarget.value)}
          error={errors.event_place}
        />
      </Group>
      {values.event_type === 'marriage' && (
        <>
          <TextInput
            label="Spouse Name"
            value={values.spouse_name}
            onChange={(e) => set('spouse_name', e.currentTarget.value)}
            error={errors.spouse_name}
          />
          <Group grow align="flex-start">
            <DateField
              label="Spouse Birth Date"
              value={values.spouse_birth_date}
              onChange={(v) => set('spouse_birth_date', v)}
              error={errors.spouse_birth_date}
            />
            <TextInput
              label="Spouse Birth Place"
              value={values.spouse_birth_place}
              onChange={(e) => set('spouse_birth_place', e.currentTarget.value)}
              error={errors.spouse_birth_place}
            />
          </Group>
        </>
      )}
      <TextInput
        label="Notes"
        value={values.notes}
        onChange={(e) => set('notes', e.currentTarget.value)}
        error={errors.notes}
      />
      <ErrorMessage
        error={error}
        shownFields={SHOWN_FIELDS.filter((f) => !(hidePerson && f === 'person_id'))}
        onReload={onReload}
      />
      <Group justify="flex-end">
        <Button variant="default" onClick={onCancel} disabled={saving}>Cancel</Button>
        <Button onClick={() => onSubmit(values)} loading={saving}>
          {isNew ? 'Add Event' : 'Save'}
        </Button>
      </Group>
    </Stack>
  );
}
