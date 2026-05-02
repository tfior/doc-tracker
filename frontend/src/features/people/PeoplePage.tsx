import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import {
  Title, Table, Text, Loader, Alert, Button, Group, Stack,
  Modal, TextInput, MultiSelect, Badge,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  listPeople, createPerson, updatePerson, deletePerson,
  addParent, removeParent, type Person, type UpdatePersonInput,
} from '../../api/people';

function fullName(p: Person) {
  return `${p.first_name} ${p.last_name}`;
}

interface PersonFormProps {
  people: Person[];
  editing: Person | null;
  onSubmit: (fields: UpdatePersonInput & { first_name: string; last_name: string }, parentIds: string[], childIds: string[]) => void;
  onClose: () => void;
  loading: boolean;
  error: string;
}

function PersonForm({ people, editing, onSubmit, onClose, loading, error }: PersonFormProps) {
  const [firstName, setFirstName] = useState(editing?.first_name ?? '');
  const [lastName, setLastName] = useState(editing?.last_name ?? '');
  const [birthDate, setBirthDate] = useState(editing?.birth_date ?? '');
  const [birthPlace, setBirthPlace] = useState(editing?.birth_place ?? '');
  const [deathDate, setDeathDate] = useState(editing?.death_date ?? '');
  const [notes, setNotes] = useState(editing?.notes ?? '');
  const [parentIds, setParentIds] = useState<string[]>(editing?.parent_ids ?? []);
  const [childIds, setChildIds] = useState<string[]>(
    editing ? people.filter((p) => (p.parent_ids ?? []).includes(editing.id)).map((p) => p.id) : [],
  );

  const otherPeople = people.filter((p) => p.id !== editing?.id);

  const personOptions = otherPeople.map((p) => ({
    value: p.id,
    label: fullName(p),
  }));

  // Parents field: exclude anyone who is already a child of this person
  const parentOptions = personOptions.filter((o) => !childIds.includes(o.value));
  // Children field: exclude anyone who is already a parent of this person
  const childOptions = personOptions.filter((o) => !parentIds.includes(o.value));

  function handleSubmit() {
    onSubmit(
      {
        first_name: firstName,
        last_name: lastName,
        birth_date: birthDate || null,
        birth_place: birthPlace || null,
        death_date: deathDate || null,
        notes: notes || null,
      },
      parentIds,
      childIds,
    );
  }

  return (
    <Stack>
      <Group grow>
        <TextInput
          label="First Name"
          value={firstName}
          onChange={(e) => setFirstName(e.currentTarget.value)}
          required
        />
        <TextInput
          label="Last Name"
          value={lastName}
          onChange={(e) => setLastName(e.currentTarget.value)}
          required
        />
      </Group>
      <Group grow>
        <TextInput
          label="Birth Date"
          placeholder="YYYY-MM-DD"
          value={birthDate}
          onChange={(e) => setBirthDate(e.currentTarget.value)}
        />
        <TextInput
          label="Birth Place"
          value={birthPlace}
          onChange={(e) => setBirthPlace(e.currentTarget.value)}
        />
      </Group>
      <Group grow>
        <TextInput
          label="Death Date"
          placeholder="YYYY-MM-DD"
          value={deathDate}
          onChange={(e) => setDeathDate(e.currentTarget.value)}
        />
      </Group>
      <TextInput
        label="Notes"
        value={notes}
        onChange={(e) => setNotes(e.currentTarget.value)}
      />
      <MultiSelect
        label="Parents"
        description="Up to 2 people from this case"
        data={parentOptions}
        value={parentIds}
        onChange={(val) => setParentIds(val.slice(0, 2))}
        maxValues={2}
        searchable
        clearable
      />
      <MultiSelect
        label="Children"
        description="People in this case who have this person as a parent"
        data={childOptions}
        value={childIds}
        onChange={setChildIds}
        searchable
        clearable
      />
      {error && <Alert color="red">{error}</Alert>}
      <Group justify="flex-end">
        <Button variant="default" onClick={onClose} disabled={loading}>Cancel</Button>
        <Button onClick={handleSubmit} loading={loading}>
          {editing ? 'Save' : 'Add Person'}
        </Button>
      </Group>
    </Stack>
  );
}

export default function PeoplePage() {
  const { caseId } = useParams<{ caseId: string }>();
  const queryClient = useQueryClient();

  const { data, isLoading, isError } = useQuery({
    queryKey: ['people', caseId],
    queryFn: () => listPeople(caseId!),
  });

  const [modalOpened, modalHandlers] = useDisclosure(false);
  const [deleteOpened, deleteHandlers] = useDisclosure(false);
  const [editingPerson, setEditingPerson] = useState<Person | null>(null);
  const [deletingPerson, setDeletingPerson] = useState<Person | null>(null);
  const [formError, setFormError] = useState('');

  const people = data?.items ?? [];

  async function syncRelationships(
    personId: string,
    newParentIds: string[],
    newChildIds: string[],
    currentParentIds: string[],
    currentChildIds: string[],
  ) {
    const parentsToAdd = newParentIds.filter((id) => !currentParentIds.includes(id));
    const parentsToRemove = currentParentIds.filter((id) => !newParentIds.includes(id));
    const childrenToAdd = newChildIds.filter((id) => !currentChildIds.includes(id));
    const childrenToRemove = currentChildIds.filter((id) => !newChildIds.includes(id));

    await Promise.all([
      ...parentsToAdd.map((pid) => addParent(caseId!, personId, pid)),
      ...parentsToRemove.map((pid) => removeParent(caseId!, personId, pid)),
      ...childrenToAdd.map((childId) => addParent(caseId!, childId, personId)),
      ...childrenToRemove.map((childId) => removeParent(caseId!, childId, personId)),
    ]);
  }

  const createMutation = useMutation({
    mutationFn: async ({
      fields,
      parentIds,
      childIds,
    }: {
      fields: UpdatePersonInput & { first_name: string; last_name: string };
      parentIds: string[];
      childIds: string[];
    }) => {
      const person = await createPerson(caseId!, fields);
      await syncRelationships(person.id, parentIds, childIds, [], []);
      return person;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['people', caseId] });
      modalHandlers.close();
    },
    onError: () => setFormError('Failed to save. Please try again.'),
  });

  const updateMutation = useMutation({
    mutationFn: async ({
      personId,
      fields,
      parentIds,
      childIds,
      currentParentIds,
      currentChildIds,
    }: {
      personId: string;
      fields: UpdatePersonInput;
      parentIds: string[];
      childIds: string[];
      currentParentIds: string[];
      currentChildIds: string[];
    }) => {
      await Promise.all([
        updatePerson(caseId!, personId, fields),
        syncRelationships(personId, parentIds, childIds, currentParentIds, currentChildIds),
      ]);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['people', caseId] });
      modalHandlers.close();
    },
    onError: () => setFormError('Failed to save. Please try again.'),
  });

  const deleteMutation = useMutation({
    mutationFn: (personId: string) => deletePerson(caseId!, personId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['people', caseId] });
      deleteHandlers.close();
      setDeletingPerson(null);
    },
    onError: () => setFormError('Failed to delete. Please try again.'),
  });

  function openCreate() {
    setEditingPerson(null);
    setFormError('');
    modalHandlers.open();
  }

  function openEdit(p: Person) {
    setEditingPerson(p);
    setFormError('');
    modalHandlers.open();
  }

  function openDelete(p: Person) {
    setDeletingPerson(p);
    setFormError('');
    deleteHandlers.open();
  }

  function handleSubmit(
    fields: UpdatePersonInput & { first_name: string; last_name: string },
    parentIds: string[],
    childIds: string[],
  ) {
    if (!fields.first_name.trim() || !fields.last_name.trim()) {
      setFormError('First and last name are required.');
      return;
    }
    setFormError('');
    if (editingPerson) {
      const currentChildIds = people
        .filter((p) => (p.parent_ids ?? []).includes(editingPerson.id))
        .map((p) => p.id);
      updateMutation.mutate({
        personId: editingPerson.id,
        fields,
        parentIds,
        childIds,
        currentParentIds: editingPerson.parent_ids ?? [],
        currentChildIds,
      });
    } else {
      createMutation.mutate({ fields, parentIds, childIds });
    }
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;

  return (
    <>
      <Group justify="space-between" mb="md">
        <Title order={2}>People</Title>
        <Button onClick={openCreate}>Add Person</Button>
      </Group>

      {isLoading && <Loader />}
      {isError && <Alert color="red">Failed to load people.</Alert>}

      {!isLoading && !isError && (
        <Table striped highlightOnHover withTableBorder>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Name</Table.Th>
              <Table.Th>Birth Date</Table.Th>
              <Table.Th>Birth Place</Table.Th>
              <Table.Th>Death Date</Table.Th>
              <Table.Th>Parents</Table.Th>
              <Table.Th />
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {people.length === 0 && (
              <Table.Tr>
                <Table.Td colSpan={6}>
                  <Text c="dimmed">No people yet.</Text>
                </Table.Td>
              </Table.Tr>
            )}
            {people.map((p) => {
              const parents = people.filter((other) => (p.parent_ids ?? []).includes(other.id));
              return (
                <Table.Tr key={p.id} onClick={() => openEdit(p)} style={{ cursor: 'pointer' }}>
                  <Table.Td>{fullName(p)}</Table.Td>
                  <Table.Td>{p.birth_date ?? '—'}</Table.Td>
                  <Table.Td>{p.birth_place ?? '—'}</Table.Td>
                  <Table.Td>{p.death_date ?? '—'}</Table.Td>
                  <Table.Td>
                    {parents.length === 0 ? (
                      <Text size="sm" c="dimmed">—</Text>
                    ) : (
                      <Group gap={4}>
                        {parents.map((parent) => (
                          <Badge key={parent.id} variant="outline" size="sm">
                            {fullName(parent)}
                          </Badge>
                        ))}
                      </Group>
                    )}
                  </Table.Td>
                  <Table.Td>
                    <Group gap="xs" justify="flex-end" onClick={(e) => e.stopPropagation()}>
                      <Button size="xs" variant="subtle" onClick={() => openDelete(p)} color="red">Delete</Button>
                    </Group>
                  </Table.Td>
                </Table.Tr>
              );
            })}
          </Table.Tbody>
        </Table>
      )}

      <Modal
        opened={modalOpened}
        onClose={modalHandlers.close}
        title={editingPerson ? `Edit — ${fullName(editingPerson)}` : 'Add Person'}
        size="lg"
      >
        <PersonForm
          people={people}
          editing={editingPerson}
          onSubmit={handleSubmit}
          onClose={modalHandlers.close}
          loading={isSaving}
          error={formError}
        />
      </Modal>

      <Modal opened={deleteOpened} onClose={deleteHandlers.close} title="Delete Person">
        <Stack>
          <Text>
            Move <strong>{deletingPerson ? fullName(deletingPerson) : ''}</strong> to trash? It can
            be restored from the trash view.
          </Text>
          {formError && <Alert color="red">{formError}</Alert>}
          <Group justify="flex-end">
            <Button variant="default" onClick={deleteHandlers.close} disabled={deleteMutation.isPending}>
              Cancel
            </Button>
            <Button
              color="red"
              loading={deleteMutation.isPending}
              onClick={() => deletingPerson && deleteMutation.mutate(deletingPerson.id)}
            >
              Move to Trash
            </Button>
          </Group>
        </Stack>
      </Modal>
    </>
  );
}
