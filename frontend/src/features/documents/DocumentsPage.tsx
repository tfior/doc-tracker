/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import { Title, Loader, Alert, Button, Group } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import type { Document } from '../../api/documents';
import { documentsQuery, documentStatusesQuery, peopleQuery } from '../../api/queries';
import ErrorMessage from '../../components/ErrorMessage';
import DocumentsTable from './DocumentsTable';
import DocumentFormModal from './DocumentFormModal';
import DocumentDeleteModal from './DocumentDeleteModal';

export default function DocumentsPage() {
  const { caseId } = useParams<{ caseId: string }>();
  const docsQuery = useQuery(documentsQuery(caseId!));
  const peopleQ = useQuery(peopleQuery(caseId!));
  const statusesQuery = useQuery(documentStatusesQuery());

  const [formOpened, formHandlers] = useDisclosure(false);
  const [deleteOpened, deleteHandlers] = useDisclosure(false);
  const [editing, setEditing] = useState<Document | null>(null);
  const [deleting, setDeleting] = useState<Document | null>(null);

  function openForm(doc: Document | null) {
    setEditing(doc);
    formHandlers.open();
  }

  function openDelete(doc: Document) {
    setDeleting(doc);
    deleteHandlers.open();
  }

  const people = peopleQ.data?.items ?? [];
  const isLoading = docsQuery.isLoading || peopleQ.isLoading || statusesQuery.isLoading;
  const error = docsQuery.error ?? peopleQ.error ?? statusesQuery.error;

  return (
    <>
      <Group justify="space-between" mb="md">
        <Title order={2}>Documents</Title>
        <Button onClick={() => openForm(null)} disabled={people.length === 0}>Add Document</Button>
      </Group>

      {peopleQ.data && people.length === 0 && (
        <Alert color="blue" mb="md">Add people to this case before adding documents.</Alert>
      )}

      {isLoading && <Loader />}
      <ErrorMessage error={error} title="Could not load documents" />

      {!isLoading && !error && (
        <DocumentsTable
          caseId={caseId!}
          documents={docsQuery.data!.items}
          people={people}
          statuses={statusesQuery.data!}
          onEdit={openForm}
          onDelete={openDelete}
        />
      )}

      <DocumentFormModal caseId={caseId!} record={editing} opened={formOpened} onClose={formHandlers.close} />
      <DocumentDeleteModal caseId={caseId!} record={deleting} opened={deleteOpened} onClose={deleteHandlers.close} />
    </>
  );
}
