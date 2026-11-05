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
import { RECORDED_FIELDS, type DocumentFormValues, type PhaseValues } from './documentFormValues';
import { fieldErrors } from '../../lib/errors';
import { formatQualifiedDate, formatTimestampDate, localDay } from '../../lib/dates';
import { personOptions } from '../people/personName';
import { eventTypeLabel } from '../life-events/options';
import {
  DOC_TYPE_OPTIONS, PHASES, PHASE_LABELS, type Phase,
  phaseReferenceField, phaseSinceField, phaseStatusField, phaseStatusOptions,
} from './options';

const SHOWN_FIELDS: string[] = [
  'person_id', 'life_event_id', 'document_type', 'title', 'issuing_authority', 'issue_date',
  'registration_date', 'recorded_given_name', 'recorded_surname', 'recorded_birth_date',
  'recorded_birth_place', 'notes', 'details_checked', 'for_submission',
  ...PHASES.flatMap((p) => [phaseStatusField(p), phaseSinceField(p), phaseReferenceField(p)]),
];

interface DocumentFormProps {
  initial: DocumentFormValues;
  /** Who checked the details, and when, as saved; null when unchecked or new. */
  checked: { by: string | null; at: string } | null;
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
  initial, checked, isNew, people, lifeEvents, statuses, hidePerson, hideLifeEvent,
  saving, error, onSubmit, onCancel, onReload,
}: DocumentFormProps) {
  const [values, setValues] = useState(initial);
  const errors = fieldErrors(error);

  function set<K extends keyof DocumentFormValues>(key: K, value: DocumentFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  // The check covers the recorded details, so editing one clears it; the user
  // ticks it again once they have compared the new value with the scan.
  // Putting every detail back as it was restores the saved check.
  function setRecorded<K extends typeof RECORDED_FIELDS[number]>(key: K, value: DocumentFormValues[K]) {
    setValues((prev) => {
      const next = { ...prev, [key]: value };
      const unchanged = RECORDED_FIELDS.every((f) => JSON.stringify(next[f]) === JSON.stringify(initial[f]));
      return { ...next, details_checked: unchanged ? initial.details_checked : false };
    });
  }

  // A status change is dated today, which the server records unless the user
  // corrects the date.
  function setPhase(phase: Phase, change: Partial<PhaseValues>) {
    setValues((prev) => {
      const p = { ...prev.phases[phase], ...change };
      if (change.status_id !== undefined && change.since === undefined && !p.since_edited) {
        p.since = change.status_id === initial.phases[phase].status_id
          ? initial.phases[phase].since
          : localDay(new Date());
      }
      return { ...prev, phases: { ...prev.phases, [phase]: p } };
    });
  }
  const today = localDay(new Date());
  const recordedChanged = RECORDED_FIELDS.some((f) => JSON.stringify(values[f]) !== JSON.stringify(initial[f]));

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
          onChange={(e) => setRecorded('recorded_given_name', e.currentTarget.value)}
          error={errors.recorded_given_name}
        />
        <TextInput
          label="Recorded Surname"
          value={values.recorded_surname}
          onChange={(e) => setRecorded('recorded_surname', e.currentTarget.value)}
          error={errors.recorded_surname}
        />
      </Group>
      <PartialDateInput
        label="Recorded Birth Date"
        value={values.recorded_birth_date}
        onChange={(v) => setRecorded('recorded_birth_date', v)}
        error={errors.recorded_birth_date}
      />
      <TextInput
        label="Recorded Birth Place"
        value={values.recorded_birth_place}
        onChange={(e) => setRecorded('recorded_birth_place', e.currentTarget.value)}
        error={errors.recorded_birth_place}
      />

      {!isNew && (
        <>
          <Divider label="Collection status" labelPosition="left" />
          {PHASES.map((phase) => (
            <Group key={phase} grow align="flex-start">
              <Select
                label={PHASE_LABELS[phase]}
                data={phaseStatusOptions(statuses, phase)}
                value={values.phases[phase].status_id}
                onChange={(v) => v && setPhase(phase, { status_id: v })}
                error={errors[phaseStatusField(phase)]}
              />
              <TextInput
                type="date"
                label="Since"
                max={today}
                value={values.phases[phase].since}
                onChange={(e) => setPhase(phase, {
                  since: e.currentTarget.value,
                  since_edited: e.currentTarget.value !== initial.phases[phase].since,
                })}
                error={errors[phaseSinceField(phase)]}
              />
              <TextInput
                label="Reference"
                placeholder="Order or case number"
                value={values.phases[phase].reference}
                onChange={(e) => setPhase(phase, { reference: e.currentTarget.value })}
                error={errors[phaseReferenceField(phase)]}
              />
            </Group>
          ))}
        </>
      )}

      <Divider label="Notes & checks" labelPosition="left" />
      <TextInput
        label="Notes"
        value={values.notes}
        onChange={(e) => set('notes', e.currentTarget.value)}
        error={errors.notes}
      />
      <Checkbox
        label="Details checked against scan"
        description={checkedDescription(values.details_checked, initial.details_checked, recordedChanged, checked)}
        checked={values.details_checked}
        onChange={(e) => set('details_checked', e.currentTarget.checked)}
        error={errors.details_checked}
      />
      <Checkbox
        label="For submission"
        description="Part of the documents submitted with an application"
        checked={values.for_submission}
        onChange={(e) => set('for_submission', e.currentTarget.checked)}
        error={errors.for_submission}
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

// Who checked the details and when, while the saved check still stands; or
// what saving will do to the check.
function checkedDescription(
  nowChecked: boolean, wasChecked: boolean, recordedChanged: boolean,
  checked: { by: string | null; at: string } | null,
): string | undefined {
  if (wasChecked && !nowChecked) return 'Saving clears the check.';
  if (!nowChecked) return undefined;
  if (recordedChanged || !checked) return 'Saving records a new check by you.';
  return `Checked${checked.by ? ` by ${checked.by}` : ''} on ${formatTimestampDate(checked.at)}`;
}
