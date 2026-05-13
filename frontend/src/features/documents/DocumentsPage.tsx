import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import {
  Title, Table, Text, Loader, Alert, Button, Group, Stack,
  Modal, Select, Badge,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { listPeople, type Person } from '../../api/people';
import {
  listDocuments, listDocumentStatuses, deleteDocument,
  transitionStatus, type Document, type DocumentStatus,
} from '../../api/documents';
import DocumentFormModal from './DocumentFormModal';

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const DOC_TYPE_OPTIONS = [
  { value: 'birth_certificate', label: 'Birth Certificate' },
  { value: 'marriage_certificate', label: 'Marriage Certificate' },
  { value: 'naturalization', label: 'Naturalization' },
  { value: 'death_certificate', label: 'Death Certificate' },
  { value: 'other', label: 'Other' },
];

const PHASES = ['official_copy', 'amendment', 'apostille', 'translation'] as const;


function docTypeLabel(type: string) {
  return DOC_TYPE_OPTIONS.find((o) => o.value === type)?.label ?? type;
}

function personName(id: string, people: Person[]) {
  const p = people.find((p) => p.id === id);
  return p ? `${p.first_name} ${p.last_name}` : '—';
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export default function DocumentsPage() {
  const { caseId } = useParams<{ caseId: string }>();
  const queryClient = useQueryClient();

  const docsQuery = useQuery({
    queryKey: ['documents', caseId],
    queryFn: () => listDocuments(caseId!),
  });

  const peopleQuery = useQuery({
    queryKey: ['people', caseId],
    queryFn: () => listPeople(caseId!),
  });

  const statusesQuery = useQuery({
    queryKey: ['document-statuses'],
    queryFn: listDocumentStatuses,
    staleTime: 5 * 60 * 1000,
  });

  const [formOpened, formHandlers] = useDisclosure(false);
  const [deleteOpened, deleteHandlers] = useDisclosure(false);
  const [editingDoc, setEditingDoc] = useState<Document | null>(null);
  const [deletingDoc, setDeletingDoc] = useState<Document | null>(null);
  const [deleteError, setDeleteError] = useState('');

  const people: Person[] = peopleQuery.data?.items ?? [];
  const docs: Document[] = docsQuery.data?.items ?? [];
  const allStatuses: DocumentStatus[] = statusesQuery.data ?? [];

  function statusesForPhase(phase: string) {
    return allStatuses
      .filter((s) => s.phase === phase || s.phase === 'any')
      .map((s) => ({ value: s.id, label: s.label }));
  }

  const inlineStatusMutation = useMutation({
    mutationFn: ({ docId, phase, statusId }: { docId: string; phase: string; statusId: string }) =>
      transitionStatus(caseId!, docId, phase, statusId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['documents', caseId] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (docId: string) => deleteDocument(caseId!, docId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['documents', caseId] });
      queryClient.invalidateQueries({ queryKey: ['life-events', caseId] });
      deleteHandlers.close();
      setDeletingDoc(null);
    },
    onError: () => setDeleteError('Failed to delete. Please try again.'),
  });

  function openCreate() {
    setEditingDoc(null);
    formHandlers.open();
  }

  function openEdit(doc: Document) {
    setEditingDoc(doc);
    formHandlers.open();
  }

  function openDelete(doc: Document) {
    setDeletingDoc(doc);
    setDeleteError('');
    deleteHandlers.open();
  }

  const isLoading = docsQuery.isLoading || peopleQuery.isLoading || statusesQuery.isLoading;
  const isError = docsQuery.isError || peopleQuery.isError;

  return (
    <>
      <Group justify="space-between" mb="md">
        <Title order={2}>Documents</Title>
        <Button onClick={openCreate} disabled={people.length === 0}>Add Document</Button>
      </Group>

      {people.length === 0 && !isLoading && (
        <Alert color="blue" mb="md">Add people to this case before adding documents.</Alert>
      )}

      {isLoading && <Loader />}
      {isError && <Alert color="red">Failed to load data.</Alert>}

      {!isLoading && !isError && (
        <Table striped withTableBorder style={{ tableLayout: 'fixed' }}>
          <Table.Thead>
            <Table.Tr>
              <Table.Th style={{ width: '18%' }}>Title</Table.Th>
              <Table.Th style={{ width: '12%' }}>Person</Table.Th>
              <Table.Th style={{ width: '10%' }}>Type</Table.Th>
              <Table.Th style={{ width: '12%' }}>Official Copy</Table.Th>
              <Table.Th style={{ width: '12%' }}>Amendment</Table.Th>
              <Table.Th style={{ width: '12%' }}>Apostille</Table.Th>
              <Table.Th style={{ width: '12%' }}>Translation</Table.Th>
              <Table.Th style={{ width: '6%' }}>Verified</Table.Th>
              <Table.Th style={{ width: '6%' }} />
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {docs.length === 0 && (
              <Table.Tr>
                <Table.Td colSpan={9}><Text c="dimmed">No documents yet.</Text></Table.Td>
              </Table.Tr>
            )}
            {docs.map((d) => (
              <Table.Tr key={d.id} onClick={() => openEdit(d)} style={{ cursor: 'pointer' }}>
                <Table.Td><Text size="sm" truncate>{d.title}</Text></Table.Td>
                <Table.Td><Text size="sm" truncate>{personName(d.person_id, people)}</Text></Table.Td>
                <Table.Td><Text size="xs" c="dimmed">{docTypeLabel(d.document_type)}</Text></Table.Td>
                {PHASES.map((phase) => {
                  const current = d[`${phase}_status` as keyof Document] as { id: string; label: string };
                  return (
                    <Table.Td key={phase} onClick={(e) => e.stopPropagation()}>
                      <Select
                        size="xs"
                        data={statusesForPhase(phase)}
                        value={current.id}
                        onChange={(v) =>
                          v && inlineStatusMutation.mutate({ docId: d.id, phase, statusId: v })
                        }
                        styles={{ input: { minHeight: 'unset', fontSize: '11px' } }}
                      />
                    </Table.Td>
                  );
                })}
                <Table.Td>
                  {d.is_verified
                    ? <Badge color="green" size="xs">Yes</Badge>
                    : <Badge color="gray" variant="outline" size="xs">No</Badge>}
                </Table.Td>
                <Table.Td onClick={(e) => e.stopPropagation()}>
                  <Button size="xs" variant="subtle" color="red" onClick={() => openDelete(d)}>
                    Delete
                  </Button>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      )}

      <DocumentFormModal
        caseId={caseId!}
        editing={editingDoc}
        opened={formOpened}
        onClose={formHandlers.close}
        onSuccess={formHandlers.close}
      />

      <Modal opened={deleteOpened} onClose={deleteHandlers.close} title="Delete Document">
        <Stack>
          <Text>
            Move <strong>{deletingDoc?.title}</strong> to trash? It can be restored from the trash
            view.
          </Text>
          {deleteError && <Alert color="red">{deleteError}</Alert>}
          <Group justify="flex-end">
            <Button variant="default" onClick={deleteHandlers.close} disabled={deleteMutation.isPending}>
              Cancel
            </Button>
            <Button
              color="red"
              loading={deleteMutation.isPending}
              onClick={() => deletingDoc && deleteMutation.mutate(deletingDoc.id)}
            >
              Move to Trash
            </Button>
          </Group>
        </Stack>
      </Modal>
    </>
  );
}
