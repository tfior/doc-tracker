/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import { Title, Loader, Button, Group } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import type { Person } from '../../api/people';
import { peopleQuery } from '../../api/queries';
import ErrorMessage from '../../components/ErrorMessage';
import PeopleTable from './PeopleTable';
import PersonFormModal from './PersonFormModal';
import PersonDeleteModal from './PersonDeleteModal';

export default function PeoplePage() {
  const { caseId } = useParams<{ caseId: string }>();
  const { data, isLoading, error } = useQuery(peopleQuery(caseId!));

  const [formOpened, formHandlers] = useDisclosure(false);
  const [deleteOpened, deleteHandlers] = useDisclosure(false);
  const [editing, setEditing] = useState<Person | null>(null);
  const [deleting, setDeleting] = useState<Person | null>(null);

  function openForm(p: Person | null) {
    setEditing(p);
    formHandlers.open();
  }

  function openDelete(p: Person) {
    setDeleting(p);
    deleteHandlers.open();
  }

  return (
    <>
      <Group justify="space-between" mb="md">
        <Title order={2}>People</Title>
        <Button onClick={() => openForm(null)}>Add Person</Button>
      </Group>

      {isLoading && <Loader />}
      <ErrorMessage error={error} title="Could not load people" />
      {data && <PeopleTable people={data.items} onEdit={openForm} onDelete={openDelete} />}

      <PersonFormModal caseId={caseId!} record={editing} opened={formOpened} onClose={formHandlers.close} />
      <PersonDeleteModal caseId={caseId!} record={deleting} opened={deleteOpened} onClose={deleteHandlers.close} />
    </>
  );
}
