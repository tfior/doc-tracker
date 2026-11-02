/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Modal, Stack, Loader } from '@mantine/core';
import { createPerson, updatePerson, type Person } from '../../api/people';
import { documentsQuery, invalidateCase, lifeEventsQuery, peopleQuery } from '../../api/queries';
import { changedFields } from '../../lib/changes';
import LifeEventListSection from '../life-events/LifeEventListSection';
import DocumentListSection from '../documents/DocumentListSection';
import PersonForm from './PersonForm';
import { personFormValues, personInput, type PersonFormValues } from './personFormValues';
import { fullName } from './personName';

interface PersonFormModalProps {
  caseId: string;
  /** The person to edit, or null to add one. */
  record: Person | null;
  opened: boolean;
  onClose: () => void;
}

export default function PersonFormModal(props: PersonFormModalProps) {
  const { caseId, record, opened, onClose } = props;
  const people = useQuery(peopleQuery(caseId)).data?.items;
  // A person's children are worked out from the people list, and are saved as
  // a full list, so the form waits for it rather than start with none.
  return (
    <Modal opened={opened} onClose={onClose} title={record ? `Edit — ${fullName(record)}` : 'Add Person'} size="lg">
      {people ? <PersonFormBody {...props} people={people} /> : <Loader />}
    </Modal>
  );
}

// The modal's content is unmounted when it closes, so each opening starts from
// the record as it was when opened, and saves against that version.
function PersonFormBody({ caseId, record, onClose, people }: PersonFormModalProps & { people: Person[] }) {
  const queryClient = useQueryClient();
  const lifeEvents = useQuery(lifeEventsQuery(caseId)).data?.items ?? [];
  const documents = useQuery(documentsQuery(caseId)).data?.items ?? [];

  // The starting values are fixed when the modal opens, and the save is sent
  // with the version read alongside them. A family link changed elsewhere since
  // raises this person's version too, so the save is refused rather than
  // silently undoing it (domain model, rule 7).
  const [base, setBase] = useState(() => ({ person: record, initial: personFormValues(record, people) }));
  const person = base.person;

  const save = useMutation({
    mutationFn: async (values: PersonFormValues) => {
      if (!person) return createPerson(caseId, personInput(values));
      const input = changedFields(personInput(base.initial), personInput(values));
      if (Object.keys(input).length === 0) return person;
      return updatePerson(caseId, person.id, person.version, input);
    },
    onSuccess: () => {
      invalidateCase(queryClient, caseId);
      onClose();
    },
  });

  async function reload() {
    const { items } = await queryClient.fetchQuery({ ...peopleQuery(caseId), staleTime: 0 });
    const fresh = items.find((p) => p.id === person!.id);
    save.reset();
    if (fresh) setBase({ person: fresh, initial: personFormValues(fresh, items) });
    else onClose();
  }

  return (
    <Stack>
      <PersonForm
        key={person?.version ?? 'new'}
        initial={base.initial}
        isNew={!person}
        others={people.filter((p) => p.id !== person?.id)}
        saving={save.isPending}
        error={save.error}
        onSubmit={(values) => save.mutate(values)}
        onCancel={onClose}
        onReload={reload}
      />
      {person && (
        <>
          <LifeEventListSection
            caseId={caseId}
            personId={person.id}
            lifeEvents={lifeEvents.filter((le) => le.person_id === person.id)}
          />
          <DocumentListSection
            caseId={caseId}
            documents={documents.filter((d) => d.person_id === person.id)}
            emptyText="No documents yet."
            personId={person.id}
          />
        </>
      )}
    </Stack>
  );
}
