import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import {
  Title, Table, Text, Loader, Alert, Button, Group, Stack,
  Modal, TextInput, MultiSelect, Badge, Divider, Select,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  listPeople, createPerson, updatePerson, deletePerson,
  addParent, removeParent, type Person, type UpdatePersonInput,
} from '../../api/people';
import {
  listLifeEvents, createLifeEvent, updateLifeEvent, deleteLifeEvent,
  type LifeEvent, type UpdateLifeEventInput,
} from '../../api/lifeevents';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function fullName(p: Person) {
  return `${p.first_name} ${p.last_name}`;
}

const EVENT_TYPE_OPTIONS = [
  { value: 'birth', label: 'Birth' },
  { value: 'marriage', label: 'Marriage' },
  { value: 'death', label: 'Death' },
  { value: 'naturalization', label: 'Naturalization' },
  { value: 'immigration', label: 'Immigration' },
  { value: 'other', label: 'Other' },
];

function eventTypeLabel(type: string) {
  return EVENT_TYPE_OPTIONS.find((o) => o.value === type)?.label ?? type;
}

// ---------------------------------------------------------------------------
// Life event form (no person select — person is contextual)
// ---------------------------------------------------------------------------

interface LifeEventFormProps {
  editing: LifeEvent | null;
  onSubmit: (fields: UpdateLifeEventInput & { event_type: string }) => void;
  onClose: () => void;
  loading: boolean;
  error: string;
}

function LifeEventForm({ editing, onSubmit, onClose, loading, error }: LifeEventFormProps) {
  const [eventType, setEventType] = useState(editing?.event_type ?? '');
  const [eventDate, setEventDate] = useState(editing?.event_date ?? '');
  const [eventPlace, setEventPlace] = useState(editing?.event_place ?? '');
  const [spouseName, setSpouseName] = useState(editing?.spouse_name ?? '');
  const [spouseBirthDate, setSpouseBirthDate] = useState(editing?.spouse_birth_date ?? '');
  const [spouseBirthPlace, setSpouseBirthPlace] = useState(editing?.spouse_birth_place ?? '');
  const [notes, setNotes] = useState(editing?.notes ?? '');

  const isMarriage = eventType === 'marriage';

  function handleSubmit() {
    onSubmit({
      event_type: eventType,
      event_date: eventDate || null,
      event_place: eventPlace || null,
      spouse_name: isMarriage ? (spouseName || null) : null,
      spouse_birth_date: isMarriage ? (spouseBirthDate || null) : null,
      spouse_birth_place: isMarriage ? (spouseBirthPlace || null) : null,
      notes: notes || null,
    });
  }

  return (
    <Stack>
      <Select
        label="Event Type"
        data={EVENT_TYPE_OPTIONS}
        value={eventType}
        onChange={(v) => setEventType(v ?? '')}
        required
      />
      <Group grow>
        <TextInput
          label="Date"
          placeholder="YYYY-MM-DD"
          value={eventDate}
          onChange={(e) => setEventDate(e.currentTarget.value)}
        />
        <TextInput
          label="Place"
          value={eventPlace}
          onChange={(e) => setEventPlace(e.currentTarget.value)}
        />
      </Group>
      {isMarriage && (
        <>
          <TextInput
            label="Spouse Name"
            value={spouseName}
            onChange={(e) => setSpouseName(e.currentTarget.value)}
          />
          <Group grow>
            <TextInput
              label="Spouse Birth Date"
              placeholder="YYYY-MM-DD"
              value={spouseBirthDate}
              onChange={(e) => setSpouseBirthDate(e.currentTarget.value)}
            />
            <TextInput
              label="Spouse Birth Place"
              value={spouseBirthPlace}
              onChange={(e) => setSpouseBirthPlace(e.currentTarget.value)}
            />
          </Group>
        </>
      )}
      <TextInput
        label="Notes"
        value={notes}
        onChange={(e) => setNotes(e.currentTarget.value)}
      />
      {error && <Alert color="red">{error}</Alert>}
      <Group justify="flex-end">
        <Button variant="default" onClick={onClose} disabled={loading}>Cancel</Button>
        <Button onClick={handleSubmit} loading={loading}>
          {editing ? 'Save' : 'Add Event'}
        </Button>
      </Group>
    </Stack>
  );
}

// ---------------------------------------------------------------------------
// Person form (fields + relationships + life events section)
// ---------------------------------------------------------------------------

interface PersonFormProps {
  people: Person[];
  editing: Person | null;
  lifeEvents: LifeEvent[];
  onSubmit: (fields: UpdatePersonInput & { first_name: string; last_name: string }, parentIds: string[], childIds: string[]) => void;
  onClose: () => void;
  onAddLifeEvent: () => void;
  onEditLifeEvent: (le: LifeEvent) => void;
  onDeleteLifeEvent: (le: LifeEvent) => void;
  deletingLEId: string | null;
  onCancelDeleteLE: () => void;
  loading: boolean;
  error: string;
}

function PersonForm({
  people, editing, lifeEvents,
  onSubmit, onClose,
  onAddLifeEvent, onEditLifeEvent, onDeleteLifeEvent,
  deletingLEId, onCancelDeleteLE,
  loading, error,
}: PersonFormProps) {
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
  const personOptions = otherPeople.map((p) => ({ value: p.id, label: fullName(p) }));
  const parentOptions = personOptions.filter((o) => !childIds.includes(o.value));
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
        <TextInput label="First Name" value={firstName} onChange={(e) => setFirstName(e.currentTarget.value)} required />
        <TextInput label="Last Name" value={lastName} onChange={(e) => setLastName(e.currentTarget.value)} required />
      </Group>
      <Group grow>
        <TextInput label="Birth Date" placeholder="YYYY-MM-DD" value={birthDate} onChange={(e) => setBirthDate(e.currentTarget.value)} />
        <TextInput label="Birth Place" value={birthPlace} onChange={(e) => setBirthPlace(e.currentTarget.value)} />
      </Group>
      <TextInput label="Death Date" placeholder="YYYY-MM-DD" value={deathDate} onChange={(e) => setDeathDate(e.currentTarget.value)} />
      <TextInput label="Notes" value={notes} onChange={(e) => setNotes(e.currentTarget.value)} />
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

      {/* Life events — only shown when editing an existing person */}
      {editing && (
        <>
          <Divider mt="xs" />
          <Group justify="space-between" align="center">
            <Text fw={500} size="sm">Life Events</Text>
            <Button size="xs" variant="light" onClick={onAddLifeEvent}>Add</Button>
          </Group>
          {lifeEvents.length === 0 && (
            <Text size="sm" c="dimmed">No life events yet.</Text>
          )}
          <Stack gap="xs">
            {lifeEvents.map((le) => (
              <Group key={le.id} justify="space-between" wrap="nowrap"
                style={{ borderBottom: '1px solid var(--mantine-color-gray-2)', paddingBottom: 8 }}
              >
                <div>
                  <Text size="sm" fw={500}>{eventTypeLabel(le.event_type)}</Text>
                  <Text size="xs" c="dimmed">
                    {[le.event_date, le.event_place].filter(Boolean).join(' · ') || 'No date or place'}
                  </Text>
                </div>
                {deletingLEId === le.id ? (
                  <Group gap="xs" wrap="nowrap">
                    <Text size="xs" c="dimmed">Delete?</Text>
                    <Button size="xs" variant="subtle" onClick={onCancelDeleteLE}>Cancel</Button>
                    <Button size="xs" color="red" onClick={() => onDeleteLifeEvent(le)}>Delete</Button>
                  </Group>
                ) : (
                  <Group gap="xs" wrap="nowrap">
                    <Button size="xs" variant="subtle" onClick={() => onEditLifeEvent(le)}>Edit</Button>
                    <Button size="xs" variant="subtle" color="red" onClick={() => onDeleteLifeEvent(le)}>Delete</Button>
                  </Group>
                )}
              </Group>
            ))}
          </Stack>
        </>
      )}
    </Stack>
  );
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export default function PeoplePage() {
  const { caseId } = useParams<{ caseId: string }>();
  const queryClient = useQueryClient();

  const { data, isLoading, isError } = useQuery({
    queryKey: ['people', caseId],
    queryFn: () => listPeople(caseId!),
  });

  const eventsQuery = useQuery({
    queryKey: ['life-events', caseId],
    queryFn: () => listLifeEvents(caseId!),
  });

  // Person modal
  const [modalOpened, modalHandlers] = useDisclosure(false);
  const [deleteOpened, deleteHandlers] = useDisclosure(false);
  const [editingPerson, setEditingPerson] = useState<Person | null>(null);
  const [deletingPerson, setDeletingPerson] = useState<Person | null>(null);
  const [formError, setFormError] = useState('');

  // Life event modal (nested inside person modal)
  const [leModalOpened, leModalHandlers] = useDisclosure(false);
  const [editingLE, setEditingLE] = useState<LifeEvent | null>(null);
  const [leFormError, setLEFormError] = useState('');
  const [deletingLEId, setDeletingLEId] = useState<string | null>(null);

  const people = data?.items ?? [];
  const allEvents = eventsQuery.data?.items ?? [];

  // Life events for the person currently being edited
  const personLifeEvents = editingPerson
    ? allEvents.filter((e) => e.person_id === editingPerson.id)
    : [];

  // ---------------------------------------------------------------------------
  // Person mutations
  // ---------------------------------------------------------------------------

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
      fields, parentIds, childIds,
    }: { fields: UpdatePersonInput & { first_name: string; last_name: string }; parentIds: string[]; childIds: string[] }) => {
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
      personId, fields, parentIds, childIds, currentParentIds, currentChildIds,
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

  // ---------------------------------------------------------------------------
  // Life event mutations
  // ---------------------------------------------------------------------------

  const leSaveMutation = useMutation({
    mutationFn: async (fields: UpdateLifeEventInput & { event_type: string }) => {
      if (editingLE) {
        return updateLifeEvent(caseId!, editingLE.id, fields);
      } else {
        return createLifeEvent(caseId!, { ...fields, person_id: editingPerson!.id });
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['life-events', caseId] });
      queryClient.invalidateQueries({ queryKey: ['cases', caseId] });
      leModalHandlers.close();
    },
    onError: () => setLEFormError('Failed to save. Please try again.'),
  });

  const leDeleteMutation = useMutation({
    mutationFn: (eventId: string) => deleteLifeEvent(caseId!, eventId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['life-events', caseId] });
      queryClient.invalidateQueries({ queryKey: ['cases', caseId] });
      setDeletingLEId(null);
    },
  });

  // ---------------------------------------------------------------------------
  // Handlers
  // ---------------------------------------------------------------------------

  function openCreate() {
    setEditingPerson(null);
    setFormError('');
    setDeletingLEId(null);
    modalHandlers.open();
  }

  function openEdit(p: Person) {
    setEditingPerson(p);
    setFormError('');
    setDeletingLEId(null);
    modalHandlers.open();
  }

  function openDelete(p: Person) {
    setDeletingPerson(p);
    setFormError('');
    deleteHandlers.open();
  }

  function openAddLifeEvent() {
    setEditingLE(null);
    setLEFormError('');
    leModalHandlers.open();
  }

  function openEditLifeEvent(le: LifeEvent) {
    setEditingLE(le);
    setLEFormError('');
    leModalHandlers.open();
  }

  function handleDeleteLifeEvent(le: LifeEvent) {
    if (deletingLEId === le.id) {
      leDeleteMutation.mutate(le.id);
    } else {
      setDeletingLEId(le.id);
    }
  }

  function handlePersonSubmit(
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

  function handleLESubmit(fields: UpdateLifeEventInput & { event_type: string }) {
    if (!fields.event_type) {
      setLEFormError('Event type is required.');
      return;
    }
    setLEFormError('');
    leSaveMutation.mutate(fields);
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
                <Table.Td colSpan={6}><Text c="dimmed">No people yet.</Text></Table.Td>
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
                          <Badge key={parent.id} variant="outline" size="sm">{fullName(parent)}</Badge>
                        ))}
                      </Group>
                    )}
                  </Table.Td>
                  <Table.Td>
                    <Group gap="xs" justify="flex-end" onClick={(e) => e.stopPropagation()}>
                      <Button size="xs" variant="subtle" color="red" onClick={() => openDelete(p)}>Delete</Button>
                    </Group>
                  </Table.Td>
                </Table.Tr>
              );
            })}
          </Table.Tbody>
        </Table>
      )}

      {/* Person create/edit modal */}
      <Modal
        opened={modalOpened}
        onClose={modalHandlers.close}
        title={editingPerson ? `Edit — ${fullName(editingPerson)}` : 'Add Person'}
        size="lg"
      >
        <PersonForm
          people={people}
          editing={editingPerson}
          lifeEvents={personLifeEvents}
          onSubmit={handlePersonSubmit}
          onClose={modalHandlers.close}
          onAddLifeEvent={openAddLifeEvent}
          onEditLifeEvent={openEditLifeEvent}
          onDeleteLifeEvent={handleDeleteLifeEvent}
          deletingLEId={deletingLEId}
          onCancelDeleteLE={() => setDeletingLEId(null)}
          loading={isSaving}
          error={formError}
        />
      </Modal>

      {/* Person delete confirmation modal */}
      <Modal opened={deleteOpened} onClose={deleteHandlers.close} title="Delete Person">
        <Stack>
          <Text>
            Move <strong>{deletingPerson ? fullName(deletingPerson) : ''}</strong> to trash?
            It can be restored from the trash view.
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

      {/* Life event create/edit modal — nested on top of person modal */}
      <Modal
        opened={leModalOpened}
        onClose={leModalHandlers.close}
        title={editingLE ? `Edit — ${eventTypeLabel(editingLE.event_type)}` : 'Add Life Event'}
        size="lg"
      >
        <LifeEventForm
          editing={editingLE}
          onSubmit={handleLESubmit}
          onClose={leModalHandlers.close}
          loading={leSaveMutation.isPending}
          error={leFormError}
        />
      </Modal>
    </>
  );
}
