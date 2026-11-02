/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { Title, Loader, Button, Group } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import type { Case } from '../../api/cases';
import { casesQuery } from '../../api/queries';
import ErrorMessage from '../../components/ErrorMessage';
import CaseTable from './CaseTable';
import CaseFormModal from './CaseFormModal';
import CaseDeleteModal from './CaseDeleteModal';

export default function CaseListPage() {
  const navigate = useNavigate();
  const { data, isLoading, error } = useQuery(casesQuery());

  const [formOpened, formHandlers] = useDisclosure(false);
  const [deleteOpened, deleteHandlers] = useDisclosure(false);
  const [editing, setEditing] = useState<Case | null>(null);
  const [deleting, setDeleting] = useState<Case | null>(null);

  function openForm(c: Case | null) {
    setEditing(c);
    formHandlers.open();
  }

  function openDelete(c: Case) {
    setDeleting(c);
    deleteHandlers.open();
  }

  return (
    <>
      <Group justify="space-between" mb="md">
        <Title order={2}>Cases</Title>
        <Button onClick={() => openForm(null)}>New Case</Button>
      </Group>

      {isLoading && <Loader />}
      <ErrorMessage error={error} title="Could not load cases" />
      {data && <CaseTable cases={data.items} onEdit={openForm} onDelete={openDelete} />}

      <CaseFormModal
        record={editing}
        opened={formOpened}
        onClose={formHandlers.close}
        onCreated={(c) => navigate(`/cases/${c.id}`)}
      />
      <CaseDeleteModal record={deleting} opened={deleteOpened} onClose={deleteHandlers.close} />
    </>
  );
}
