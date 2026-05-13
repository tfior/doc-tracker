import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import {
  Title, Table, Text, Loader, Alert, Button, Group, Stack,
  Modal, Select, Textarea, Badge,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { listPeople, type Person } from '../../api/people';
import {
  listClaimLines, createClaimLine, updateClaimLine, deleteClaimLine,
  type ClaimLine,
} from '../../api/claimlines';

const STATUS_OPTIONS = [
  { value: 'not_yet_researched', label: 'Not Yet Researched' },
  { value: 'researching',        label: 'Researching' },
  { value: 'paused',             label: 'Paused' },
  { value: 'ineligible',         label: 'Ineligible' },
  { value: 'eligible',           label: 'Eligible' },
];

const STATUS_COLOR: Record<string, string> = {
  not_yet_researched: 'gray',
  researching:        'blue',
  paused:             'orange',
  ineligible:         'red',
  eligible:           'green',
};

function statusLabel(status: string) {
  return STATUS_OPTIONS.find((o) => o.value === status)?.label ?? status;
}

function personName(id: string, people: Person[]) {
  const p = people.find((p) => p.id === id);
  return p ? `${p.first_name} ${p.last_name}` : '—';
}

export default function ClaimLinesPage() {
  const { caseId } = useParams<{ caseId: string }>();
  const queryClient = useQueryClient();

  const linesQuery = useQuery({
    queryKey: ['claim-lines', caseId],
    queryFn: () => listClaimLines(caseId!),
  });

  const peopleQuery = useQuery({
    queryKey: ['people', caseId],
    queryFn: () => listPeople(caseId!),
  });

  const [modalOpened, modalHandlers] = useDisclosure(false);
  const [deleteOpened, deleteHandlers] = useDisclosure(false);
  const [editingLine, setEditingLine] = useState<ClaimLine | null>(null);
  const [deletingLine, setDeletingLine] = useState<ClaimLine | null>(null);
  const [formError, setFormError] = useState('');

  // Form state
  const [rootPersonId, setRootPersonId] = useState('');
  const [status, setStatus] = useState('not_yet_researched');
  const [notes, setNotes] = useState('');

  const people: Person[] = peopleQuery.data?.items ?? [];
  const lines: ClaimLine[] = linesQuery.data?.items ?? [];

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['claim-lines', caseId] });
    queryClient.invalidateQueries({ queryKey: ['cases', caseId] });
  };

  const createMutation = useMutation({
    mutationFn: () =>
      createClaimLine(caseId!, {
        root_person_id: rootPersonId,
        status,
        notes: notes.trim() || null,
      }),
    onSuccess: () => { invalidate(); modalHandlers.close(); },
    onError: () => setFormError('Failed to save. Please try again.'),
  });

  const updateMutation = useMutation({
    mutationFn: () =>
      updateClaimLine(caseId!, editingLine!.id, {
        status,
        notes: notes.trim() || null,
      }),
    onSuccess: () => { invalidate(); modalHandlers.close(); },
    onError: () => setFormError('Failed to save. Please try again.'),
  });

  const deleteMutation = useMutation({
    mutationFn: (lineId: string) => deleteClaimLine(caseId!, lineId),
    onSuccess: () => {
      invalidate();
      deleteHandlers.close();
      setDeletingLine(null);
    },
    onError: () => setFormError('Failed to delete. Please try again.'),
  });

  function openCreate() {
    setEditingLine(null);
    setRootPersonId('');
    setStatus('not_yet_researched');
    setNotes('');
    setFormError('');
    modalHandlers.open();
  }

  function openEdit(line: ClaimLine) {
    setEditingLine(line);
    setStatus(line.status);
    setNotes(line.notes ?? '');
    setFormError('');
    modalHandlers.open();
  }

  function openDelete(line: ClaimLine) {
    setDeletingLine(line);
    setFormError('');
    deleteHandlers.open();
  }

  function handleSubmit() {
    if (!editingLine && !rootPersonId) {
      setFormError('Root person is required.');
      return;
    }
    setFormError('');
    if (editingLine) {
      updateMutation.mutate();
    } else {
      createMutation.mutate();
    }
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;
  const isLoading = linesQuery.isLoading || peopleQuery.isLoading;
  const isError = linesQuery.isError || peopleQuery.isError;

  const personOptions = people.map((p) => ({
    value: p.id,
    label: `${p.first_name} ${p.last_name}`,
  }));

  return (
    <>
      <Group justify="space-between" mb="md">
        <Title order={2}>Claim Lines</Title>
        <Button onClick={openCreate} disabled={people.length === 0}>Add Claim Line</Button>
      </Group>

      {people.length === 0 && !isLoading && (
        <Alert color="blue" mb="md">Add people to this case before creating claim lines.</Alert>
      )}

      {isLoading && <Loader />}
      {isError && <Alert color="red">Failed to load data.</Alert>}

      {!isLoading && !isError && (
        <Table striped highlightOnHover withTableBorder>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Root Person</Table.Th>
              <Table.Th>Status</Table.Th>
              <Table.Th>Notes</Table.Th>
              <Table.Th />
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {lines.length === 0 && (
              <Table.Tr>
                <Table.Td colSpan={4}>
                  <Text c="dimmed">No claim lines yet.</Text>
                </Table.Td>
              </Table.Tr>
            )}
            {lines.map((line) => (
              <Table.Tr key={line.id} onClick={() => openEdit(line)} style={{ cursor: 'pointer' }}>
                <Table.Td>{personName(line.root_person_id, people)}</Table.Td>
                <Table.Td>
                  <Badge color={STATUS_COLOR[line.status] ?? 'gray'}>
                    {statusLabel(line.status)}
                  </Badge>
                </Table.Td>
                <Table.Td>
                  <Text size="sm" c="dimmed" lineClamp={1}>{line.notes ?? '—'}</Text>
                </Table.Td>
                <Table.Td onClick={(e) => e.stopPropagation()}>
                  <Group gap="xs" justify="flex-end">
                    <Button size="xs" variant="subtle" color="red" onClick={() => openDelete(line)}>
                      Delete
                    </Button>
                  </Group>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      )}

      {/* Create / Edit modal */}
      <Modal
        opened={modalOpened}
        onClose={modalHandlers.close}
        title={editingLine ? 'Edit Claim Line' : 'Add Claim Line'}
      >
        <Stack>
          {!editingLine && (
            <Select
              label="Root Person"
              description="The lineage-relevant ancestor this claim line traces from"
              data={personOptions}
              value={rootPersonId}
              onChange={(v) => setRootPersonId(v ?? '')}
              searchable
              required
            />
          )}
          {editingLine && (
            <Text size="sm" c="dimmed">
              Root person: <strong>{personName(editingLine.root_person_id, people)}</strong>
            </Text>
          )}
          <Select
            label="Status"
            data={STATUS_OPTIONS}
            value={status}
            onChange={(v) => setStatus(v ?? 'not_yet_researched')}
          />
          <Textarea
            label="Notes"
            value={notes}
            onChange={(e) => setNotes(e.currentTarget.value)}
            autosize
            minRows={2}
          />
          {formError && <Alert color="red">{formError}</Alert>}
          <Group justify="flex-end">
            <Button variant="default" onClick={modalHandlers.close} disabled={isSaving}>
              Cancel
            </Button>
            <Button onClick={handleSubmit} loading={isSaving}>
              {editingLine ? 'Save' : 'Add'}
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Delete confirmation */}
      <Modal opened={deleteOpened} onClose={deleteHandlers.close} title="Delete Claim Line">
        <Stack>
          <Text>
            Move this claim line (rooted at{' '}
            <strong>{deletingLine ? personName(deletingLine.root_person_id, people) : ''}</strong>)
            to trash? It can be restored from the trash view.
          </Text>
          {formError && <Alert color="red">{formError}</Alert>}
          <Group justify="flex-end">
            <Button variant="default" onClick={deleteHandlers.close} disabled={deleteMutation.isPending}>
              Cancel
            </Button>
            <Button
              color="red"
              loading={deleteMutation.isPending}
              onClick={() => deletingLine && deleteMutation.mutate(deletingLine.id)}
            >
              Move to Trash
            </Button>
          </Group>
        </Stack>
      </Modal>
    </>
  );
}
