import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link, useNavigate } from 'react-router-dom';
import {
  Badge, Table, Title, Text, Loader, Alert, Button, Group,
  Modal, TextInput, Select, Stack,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { listCases, createCase, updateCase, deleteCase, type Case } from '../../api/cases';

const STATUS_COLOR: Record<string, string> = {
  active: 'green',
  archived: 'gray',
  complete: 'blue',
};

const STATUS_OPTIONS = [
  { value: 'active', label: 'Active' },
  { value: 'archived', label: 'Archived' },
  { value: 'complete', label: 'Complete' },
];

function statusLabel(s: string) {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

export default function CaseListPage() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const { data, isLoading, isError } = useQuery({
    queryKey: ['cases'],
    queryFn: listCases,
  });

  // Modal state
  const [modalOpened, modalHandlers] = useDisclosure(false);
  const [deleteOpened, deleteHandlers] = useDisclosure(false);
  const [editingCase, setEditingCase] = useState<Case | null>(null);
  const [deletingCase, setDeletingCase] = useState<Case | null>(null);

  // Form state
  const [title, setTitle] = useState('');
  const [status, setStatus] = useState('active');
  const [formError, setFormError] = useState('');

  const createMutation = useMutation({
    mutationFn: createCase,
    onSuccess: (newCase) => {
      queryClient.invalidateQueries({ queryKey: ['cases'] });
      modalHandlers.close();
      navigate(`/cases/${newCase.id}`);
    },
    onError: () => setFormError('Failed to create case. Please try again.'),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, input }: { id: string; input: { title?: string; status?: string } }) =>
      updateCase(id, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cases'] });
      modalHandlers.close();
    },
    onError: () => setFormError('Failed to update case. Please try again.'),
  });

  const deleteMutation = useMutation({
    mutationFn: deleteCase,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cases'] });
      deleteHandlers.close();
      setDeletingCase(null);
    },
    onError: () => setFormError('Failed to delete case. Please try again.'),
  });

  function openCreate() {
    setEditingCase(null);
    setTitle('');
    setStatus('active');
    setFormError('');
    modalHandlers.open();
  }

  function openEdit(c: Case) {
    setEditingCase(c);
    setTitle(c.title);
    setStatus(c.status);
    setFormError('');
    modalHandlers.open();
  }

  function openDelete(c: Case) {
    setDeletingCase(c);
    setFormError('');
    deleteHandlers.open();
  }

  function handleSubmit() {
    if (!title.trim()) {
      setFormError('Title is required.');
      return;
    }
    setFormError('');
    if (editingCase) {
      updateMutation.mutate({ id: editingCase.id, input: { title: title.trim(), status } });
    } else {
      createMutation.mutate(title.trim());
    }
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;
  const cases: Case[] = data?.items ?? [];

  return (
    <>
      <Group justify="space-between" mb="md">
        <Title order={2}>Cases</Title>
        <Button onClick={openCreate}>New Case</Button>
      </Group>

      {isLoading && <Loader />}
      {isError && <Alert color="red">Failed to load cases.</Alert>}

      {!isLoading && !isError && (
        <Table striped highlightOnHover withTableBorder>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Title</Table.Th>
              <Table.Th>Status</Table.Th>
              <Table.Th />
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {cases.length === 0 && (
              <Table.Tr>
                <Table.Td colSpan={3}>
                  <Text c="dimmed">No cases yet.</Text>
                </Table.Td>
              </Table.Tr>
            )}
            {cases.map((c) => (
              <Table.Tr key={c.id}>
                <Table.Td>
                  <Link to={`/cases/${c.id}`}>{c.title}</Link>
                </Table.Td>
                <Table.Td>
                  <Badge color={STATUS_COLOR[c.status] ?? 'gray'}>
                    {statusLabel(c.status)}
                  </Badge>
                </Table.Td>
                <Table.Td>
                  <Group gap="xs" justify="flex-end">
                    <Button size="xs" variant="subtle" onClick={() => openEdit(c)}>
                      Edit
                    </Button>
                    <Button size="xs" variant="subtle" color="red" onClick={() => openDelete(c)}>
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
        title={editingCase ? 'Edit Case' : 'New Case'}
      >
        <Stack>
          <TextInput
            label="Title"
            placeholder="Case title"
            value={title}
            onChange={(e) => setTitle(e.currentTarget.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSubmit()}
            required
          />
          {editingCase && (
            <Select
              label="Status"
              data={STATUS_OPTIONS}
              value={status}
              onChange={(v) => setStatus(v ?? 'active')}
            />
          )}
          {formError && <Alert color="red">{formError}</Alert>}
          <Group justify="flex-end">
            <Button variant="default" onClick={modalHandlers.close} disabled={isSaving}>
              Cancel
            </Button>
            <Button onClick={handleSubmit} loading={isSaving}>
              {editingCase ? 'Save' : 'Create'}
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Delete confirmation modal */}
      <Modal
        opened={deleteOpened}
        onClose={deleteHandlers.close}
        title="Delete Case"
      >
        <Stack>
          <Text>
            Move <strong>{deletingCase?.title}</strong> to trash? It can be restored from the{' '}
            <strong>Trash</strong> section in the sidebar.
          </Text>
          {formError && <Alert color="red">{formError}</Alert>}
          <Group justify="flex-end">
            <Button variant="default" onClick={deleteHandlers.close} disabled={deleteMutation.isPending}>
              Cancel
            </Button>
            <Button
              color="red"
              loading={deleteMutation.isPending}
              onClick={() => deletingCase && deleteMutation.mutate(deletingCase.id)}
            >
              Move to Trash
            </Button>
          </Group>
        </Stack>
      </Modal>
    </>
  );
}
