/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { Stack, Group, TextInput, Select, Checkbox, Divider, Button } from '@mantine/core';
import type { DocumentStatus } from '../../api/documents';
import type { LifeEvent } from '../../api/lifeevents';
import type { Person } from '../../api/people';
import DateField from '../../components/DateField';
import ErrorMessage from '../../components/ErrorMessage';
import type { DocumentFormValues } from './documentFormValues';
import { fieldErrors } from '../../lib/errors';
import { personOptions } from '../people/personName';
import { eventTypeLabel } from '../life-events/options';
import {
  DOC_TYPE_OPTIONS, PHASES, PHASE_LABELS,
  phaseStatusField, phaseStatusOptions,
} from './options';

const SHOWN_FIELDS: string[] = [
  'person_id', 'life_event_id', 'document_type', 'title', 'issuing_authority', 'issue_date',
  'recorded_date', 'recorded_given_name', 'recorded_surname', 'recorded_birth_date',
  'recorded_birth_place', 'notes', 'is_verified', ...PHASES.map(phaseStatusField),
];

interface DocumentFormProps {
  initial: DocumentFormValues;
  isNew: boolean;
  people: Person[];
  lifeEvents: LifeEvent[];
  statuses: DocumentStatus[];
  /** Hides the person field; the document stays with its person. */
  hidePerson: boolean;
  /** Hides the life event field; the document stays with its event. */
  hideLifeEvent: boolean;
  saving: boolean;
  error: unknown;
  onSubmit: (values: DocumentFormValues) => void;
  onCancel: () => void;
  onReload: () => Promise<void>;
}

export default function DocumentForm({
  initial, isNew, people, lifeEvents, statuses, hidePerson, hideLifeEvent,
  saving, error, onSubmit, onCancel, onReload,
}: DocumentFormProps) {
  const [values, setValues] = useState(initial);
  const errors = fieldErrors(error);

  function set<K extends keyof DocumentFormValues>(key: K, value: DocumentFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  // A document's life event must belong to its person, so changing the person
  // clears an event that belongs to someone else.
  function setPerson(personId: string) {
    setValues((prev) => {
      const eventStillFits = lifeEvents.some((le) => le.id === prev.life_event_id && le.person_id === personId);
      return { ...prev, person_id: personId, life_event_id: eventStillFits ? prev.life_event_id : '' };
    });
  }

  const lifeEventOptions = [
    { value: '', label: 'None (ungrouped)' },
    ...lifeEvents
      .filter((le) => le.person_id === values.person_id)
      .map((le) => ({
        value: le.id,
        label: `${eventTypeLabel(le.event_type)}${le.event_date ? ` — ${le.event_date}` : ''}`,
      })),
  ];

  return (
    <Stack>
      {!hidePerson && (
        <Select
          label="Person"
          data={personOptions(people)}
          value={values.person_id}
          onChange={(v) => setPerson(v ?? '')}
          error={errors.person_id}
          searchable
          required
        />
      )}
      <Group grow align="flex-start">
        <Select
          label="Type"
          data={DOC_TYPE_OPTIONS}
          value={values.document_type}
          onChange={(v) => set('document_type', v ?? '')}
          error={errors.document_type}
          required
        />
        <TextInput
          label="Title"
          value={values.title}
          onChange={(e) => set('title', e.currentTarget.value)}
          error={errors.title}
          required
        />
      </Group>

      {!hideLifeEvent && (
        <Select
          label="Life Event"
          description="Optional — groups this document under one of the person's events"
          data={lifeEventOptions}
          value={values.life_event_id}
          onChange={(v) => set('life_event_id', v ?? '')}
          error={errors.life_event_id}
          disabled={!values.person_id}
        />
      )}

      <Divider label="Document details" labelPosition="left" />
      <Group grow align="flex-start">
        <TextInput
          label="Issuing Authority"
          value={values.issuing_authority}
          onChange={(e) => set('issuing_authority', e.currentTarget.value)}
          error={errors.issuing_authority}
        />
        <DateField
          label="Issue Date"
          value={values.issue_date}
          onChange={(v) => set('issue_date', v)}
          error={errors.issue_date}
        />
      </Group>
      <DateField
        label="Date Recorded"
        value={values.recorded_date}
        onChange={(v) => set('recorded_date', v)}
        error={errors.recorded_date}
      />

      <Divider label="Recorded information" labelPosition="left" />
      <Group grow align="flex-start">
        <TextInput
          label="Recorded Given Name"
          value={values.recorded_given_name}
          onChange={(e) => set('recorded_given_name', e.currentTarget.value)}
          error={errors.recorded_given_name}
        />
        <TextInput
          label="Recorded Surname"
          value={values.recorded_surname}
          onChange={(e) => set('recorded_surname', e.currentTarget.value)}
          error={errors.recorded_surname}
        />
      </Group>
      <Group grow align="flex-start">
        <DateField
          label="Recorded Birth Date"
          value={values.recorded_birth_date}
          onChange={(v) => set('recorded_birth_date', v)}
          error={errors.recorded_birth_date}
        />
        <TextInput
          label="Recorded Birth Place"
          value={values.recorded_birth_place}
          onChange={(e) => set('recorded_birth_place', e.currentTarget.value)}
          error={errors.recorded_birth_place}
        />
      </Group>

      {!isNew && (
        <>
          <Divider label="Collection status" labelPosition="left" />
          {PHASES.map((phase) => (
            <Select
              key={phase}
              label={PHASE_LABELS[phase]}
              data={phaseStatusOptions(statuses, phase)}
              value={values[phaseStatusField(phase)]}
              onChange={(v) => v && set(phaseStatusField(phase), v)}
              error={errors[phaseStatusField(phase)]}
            />
          ))}
        </>
      )}

      <Divider label="Notes & verification" labelPosition="left" />
      <TextInput
        label="Notes"
        value={values.notes}
        onChange={(e) => set('notes', e.currentTarget.value)}
        error={errors.notes}
      />
      <Checkbox
        label="Verified"
        checked={values.is_verified}
        onChange={(e) => set('is_verified', e.currentTarget.checked)}
        error={errors.is_verified}
      />

      <ErrorMessage
        error={error}
        shownFields={SHOWN_FIELDS.filter((f) =>
          !(hidePerson && f === 'person_id') && !(hideLifeEvent && f === 'life_event_id'))}
        onReload={onReload}
      />
      <Group justify="flex-end">
        <Button variant="default" onClick={onCancel} disabled={saving}>Cancel</Button>
        <Button onClick={() => onSubmit(values)} loading={saving}>
          {isNew ? 'Add Document' : 'Save'}
        </Button>
      </Group>
    </Stack>
  );
}
