/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Modal, Stack } from '@mantine/core';
import { createLifeEvent, updateLifeEvent, type LifeEvent } from '../../api/lifeevents';
import { documentsQuery, invalidateCase, lifeEventsQuery, peopleQuery } from '../../api/queries';
import { changedFields } from '../../lib/changes';
import { eventPeople } from '../people/personName';
import DocumentListSection from '../documents/DocumentListSection';
import LifeEventForm from './LifeEventForm';
import { lifeEventFormValues, lifeEventInput, type LifeEventFormValues } from './lifeEventFormValues';
import { eventTypeLabel } from './options';

interface LifeEventFormModalProps {
  caseId: string;
  /** The event to edit, or null to add one. */
  record: LifeEvent | null;
  opened: boolean;
  onClose: () => void;
  /** When set, the person field is hidden and a new event belongs to this person. */
  lockedPersonId?: string;
  /** Called after a save. */
  onSaved?: () => void;
}

export default function LifeEventFormModal(props: LifeEventFormModalProps) {
  const { caseId, record, opened, onClose } = props;
  const people = useQuery(peopleQuery(caseId)).data?.items ?? [];
  const title = record
    ? `${eventPeople(record, people)} — ${eventTypeLabel(record.event_type)}`
    : 'Add Life Event';

  return (
    <Modal opened={opened} onClose={onClose} title={title} size="lg">
      <LifeEventFormBody {...props} />
    </Modal>
  );
}

// The modal's content is unmounted when it closes, so each opening starts from
// the record as it was when opened, and saves against that version.
function LifeEventFormBody({ caseId, record, onClose, lockedPersonId, onSaved }: LifeEventFormModalProps) {
  const queryClient = useQueryClient();
  const people = useQuery(peopleQuery(caseId)).data?.items ?? [];
  const documents = useQuery(documentsQuery(caseId)).data?.items ?? [];
  const [base, setBase] = useState(record);

  const save = useMutation({
    mutationFn: async (values: LifeEventFormValues) => {
      if (!base) return createLifeEvent(caseId, lifeEventInput(values));
      const input = changedFields(lifeEventInput(lifeEventFormValues(base)), lifeEventInput(values));
      if (Object.keys(input).length === 0) return base;
      return updateLifeEvent(caseId, base.id, base.version, input);
    },
    onSuccess: () => {
      invalidateCase(queryClient, caseId);
      onSaved?.();
      onClose();
    },
  });

  async function reload() {
    const { items } = await queryClient.fetchQuery({ ...lifeEventsQuery(caseId), staleTime: 0 });
    const fresh = items.find((le) => le.id === base!.id);
    save.reset();
    if (fresh) setBase(fresh);
    else onClose();
  }

  return (
    <Stack>
      <LifeEventForm
        key={base?.version ?? 'new'}
        initial={lifeEventFormValues(base, lockedPersonId)}
        isNew={!base}
        people={people}
        hidePerson={!!lockedPersonId}
        saving={save.isPending}
        error={save.error}
        onSubmit={(values) => save.mutate(values)}
        onCancel={onClose}
        onReload={reload}
      />
      {base && (
        <DocumentListSection
          caseId={caseId}
          documents={documents.filter((d) => d.life_event_id === base.id)}
          emptyText="No documents linked to this event yet."
          personId={base.person_id}
          lifeEventId={base.id}
        />
      )}
    </Stack>
  );
}
