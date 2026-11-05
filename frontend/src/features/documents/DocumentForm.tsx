/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { Stack, Group, TextInput, Select, Checkbox, Divider, Button } from '@mantine/core';
import type { DocumentStatus } from '../../api/documents';
import { isEventOf, type LifeEvent } from '../../api/lifeevents';
import type { Person } from '../../api/people';
import PartialDateInput from '../../components/PartialDateInput';
import ErrorMessage from '../../components/ErrorMessage';
import type { DocumentFormValues } from './documentFormValues';
import { fieldErrors } from '../../lib/errors';
import { formatQualifiedDate } from '../../lib/dates';
import { personOptions } from '../people/personName';
import { eventTypeLabel } from '../life-events/options';
import {
  DOC_TYPE_OPTIONS, PHASES, PHASE_LABELS,
  phaseStatusField, phaseStatusOptions,
} from './options';

const SHOWN_FIELDS: string[] = [
  'person_id', 'life_event_id', 'document_type', 'title', 'issuing_authority', 'issue_date',
  'registration_date', 'recorded_given_name', 'recorded_surname', 'recorded_birth_date',
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

  // A document belongs to its life event's person. A shared marriage is one of
  // both spouses' events, so choosing it from the other spouse gives the
  // document the marriage's person.
  function setLifeEvent(eventId: string) {
    const le = lifeEvents.find((le) => le.id === eventId);
    const ungrouped = hidePerson ? initial.person_id : undefined;
    setValues((prev) => ({ ...prev, life_event_id: eventId, person_id: le?.person_id ?? ungrouped ?? prev.person_id }));
  }

  // With the person hidden, the form is opened from that person, whose events
  // stay on offer even after a shared marriage changes the document's person.
  const eventsOf = hidePerson ? initial.person_id : values.person_id;
  const lifeEventOptions = [
    { value: '', label: 'None (ungrouped)' },
    ...lifeEvents
      .filter((le) => isEventOf(le, eventsOf) || le.id === values.life_event_id)
      .map((le) => ({
        value: le.id,
        label: `${eventTypeLabel(le.event_type)}${le.event_date ? ` — ${formatQualifiedDate(le.event_date)}` : ''}`,
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
          onChange={(v) => setLifeEvent(v ?? '')}
          error={errors.life_event_id}
          disabled={!values.person_id}
        />
      )}

      <Divider label="Document details" labelPosition="left" />
      <TextInput
        label="Issuing Authority"
        value={values.issuing_authority}
        onChange={(e) => set('issuing_authority', e.currentTarget.value)}
        error={errors.issuing_authority}
      />
      <Group align="flex-start">
        <PartialDateInput
          label="Issue Date"
          value={values.issue_date}
          onChange={(v) => set('issue_date', v)}
          error={errors.issue_date}
        />
        <PartialDateInput
          label="Registration Date"
          value={values.registration_date}
          onChange={(v) => set('registration_date', v)}
          error={errors.registration_date}
        />
      </Group>

      <Divider label="As recorded in document" labelPosition="left" />
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
      <PartialDateInput
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
