/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import { Title, Loader, Alert, Button, Group } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import type { LifeEvent } from '../../api/lifeevents';
import { lifeEventsQuery, peopleQuery } from '../../api/queries';
import ErrorMessage from '../../components/ErrorMessage';
import LifeEventsTable from './LifeEventsTable';
import LifeEventFormModal from './LifeEventFormModal';
import LifeEventDeleteModal from './LifeEventDeleteModal';

export default function LifeEventsPage() {
  const { caseId } = useParams<{ caseId: string }>();
  const peopleQ = useQuery(peopleQuery(caseId!));
  const eventsQuery = useQuery(lifeEventsQuery(caseId!));

  const [formOpened, formHandlers] = useDisclosure(false);
  const [deleteOpened, deleteHandlers] = useDisclosure(false);
  const [editing, setEditing] = useState<LifeEvent | null>(null);
  const [deleting, setDeleting] = useState<LifeEvent | null>(null);

  function openForm(le: LifeEvent | null) {
    setEditing(le);
    formHandlers.open();
  }

  function openDelete(le: LifeEvent) {
    setDeleting(le);
    deleteHandlers.open();
  }

  const people = peopleQ.data?.items ?? [];
  const isLoading = eventsQuery.isLoading || peopleQ.isLoading;
  const error = eventsQuery.error ?? peopleQ.error;

  return (
    <>
      <Group justify="space-between" mb="md">
        <Title order={2}>Life Events</Title>
        <Button onClick={() => openForm(null)} disabled={people.length === 0}>Add Life Event</Button>
      </Group>

      {peopleQ.data && people.length === 0 && (
        <Alert color="blue" mb="md">Add people to this case before creating life events.</Alert>
      )}

      {isLoading && <Loader />}
      <ErrorMessage error={error} title="Could not load life events" />

      {!isLoading && !error && (
        <LifeEventsTable lifeEvents={eventsQuery.data!.items} people={people} onEdit={openForm} onDelete={openDelete} />
      )}

      <LifeEventFormModal caseId={caseId!} record={editing} opened={formOpened} onClose={formHandlers.close} />
      <LifeEventDeleteModal caseId={caseId!} record={deleting} opened={deleteOpened} onClose={deleteHandlers.close} />
    </>
  );
}
