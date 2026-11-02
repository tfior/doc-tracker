/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import { Title, Loader, Alert, Button, Group } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import type { ClaimLine } from '../../api/claimlines';
import { claimLinesQuery, peopleQuery } from '../../api/queries';
import ErrorMessage from '../../components/ErrorMessage';
import ClaimLinesTable from './ClaimLinesTable';
import ClaimLineFormModal from './ClaimLineFormModal';
import ClaimLineDeleteModal from './ClaimLineDeleteModal';

export default function ClaimLinesPage() {
  const { caseId } = useParams<{ caseId: string }>();
  const linesQuery = useQuery(claimLinesQuery(caseId!));
  const peopleQ = useQuery(peopleQuery(caseId!));

  const [formOpened, formHandlers] = useDisclosure(false);
  const [deleteOpened, deleteHandlers] = useDisclosure(false);
  const [editing, setEditing] = useState<ClaimLine | null>(null);
  const [deleting, setDeleting] = useState<ClaimLine | null>(null);

  function openForm(line: ClaimLine | null) {
    setEditing(line);
    formHandlers.open();
  }

  function openDelete(line: ClaimLine) {
    setDeleting(line);
    deleteHandlers.open();
  }

  const people = peopleQ.data?.items ?? [];
  const isLoading = linesQuery.isLoading || peopleQ.isLoading;
  const error = linesQuery.error ?? peopleQ.error;

  return (
    <>
      <Group justify="space-between" mb="md">
        <Title order={2}>Claim Lines</Title>
        <Button onClick={() => openForm(null)} disabled={people.length === 0}>Add Claim Line</Button>
      </Group>

      {peopleQ.data && people.length === 0 && (
        <Alert color="blue" mb="md">Add people to this case before creating claim lines.</Alert>
      )}

      {isLoading && <Loader />}
      <ErrorMessage error={error} title="Could not load claim lines" />

      {!isLoading && !error && (
        <ClaimLinesTable lines={linesQuery.data!.items} people={people} onEdit={openForm} onDelete={openDelete} />
      )}

      <ClaimLineFormModal caseId={caseId!} record={editing} opened={formOpened} onClose={formHandlers.close} />
      <ClaimLineDeleteModal caseId={caseId!} record={deleting} opened={deleteOpened} onClose={deleteHandlers.close} />
    </>
  );
}
