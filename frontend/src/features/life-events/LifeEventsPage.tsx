import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import {
  Title, Table, Text, Loader, Alert, Button, Group, Stack,
  Modal, TextInput, Select, Badge, Divider,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { listPeople, type Person } from '../../api/people';
import {
  listLifeEvents, createLifeEvent, updateLifeEvent,
  deleteLifeEvent, reassignLifeEvent,
  type LifeEvent, type CreateLifeEventInput, type UpdateLifeEventInput,
} from '../../api/lifeevents';
import { listDocuments, deleteDocument, type Document } from '../../api/documents';
import DocumentFormModal from '../documents/DocumentFormModal';

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

function personName(id: string, people: Person[]) {
  const p = people.find((p) => p.id === id);
  return p ? `${p.first_name} ${p.last_name}` : '—';
}

interface LifeEventFormProps {
  people: Person[];
  editing: LifeEvent | null;
  onSubmit: (personId: string, fields: CreateLifeEventInput | UpdateLifeEventInput) => void;
  onClose: () => void;
  loading: boolean;
  error: string;
}

function LifeEventForm({ people, editing, onSubmit, onClose, loading, error }: LifeEventFormProps) {
  const [personId, setPersonId] = useState(editing?.person_id ?? '');
  const [eventType, setEventType] = useState(editing?.event_type ?? '');
  const [eventDate, setEventDate] = useState(editing?.event_date ?? '');
  const [eventPlace, setEventPlace] = useState(editing?.event_place ?? '');
  const [spouseName, setSpouseName] = useState(editing?.spouse_name ?? '');
  const [spouseBirthDate, setSpouseBirthDate] = useState(editing?.spouse_birth_date ?? '');
  const [spouseBirthPlace, setSpouseBirthPlace] = useState(editing?.spouse_birth_place ?? '');
  const [notes, setNotes] = useState(editing?.notes ?? '');

  const isMarriage = eventType === 'marriage';

  const personOptions = people.map((p) => ({
    value: p.id,
    label: `${p.first_name} ${p.last_name}`,
  }));

  function handleSubmit() {
    onSubmit(personId, {
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
        label="Person"
        data={personOptions}
        value={personId}
        onChange={(v) => setPersonId(v ?? '')}
        searchable
        required
      />
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

export default function LifeEventsPage() {
  const { caseId } = useParams<{ caseId: string }>();
  const queryClient = useQueryClient();

  const peopleQuery = useQuery({
    queryKey: ['people', caseId],
    queryFn: () => listPeople(caseId!),
  });

  const eventsQuery = useQuery({
    queryKey: ['life-events', caseId],
    queryFn: () => listLifeEvents(caseId!),
  });

  const docsQuery = useQuery({
    queryKey: ['documents', caseId],
    queryFn: () => listDocuments(caseId!),
  });

  const [modalOpened, modalHandlers] = useDisclosure(false);
  const [deleteOpened, deleteHandlers] = useDisclosure(false);
  const [editingEvent, setEditingEvent] = useState<LifeEvent | null>(null);
  const [deletingEvent, setDeletingEvent] = useState<LifeEvent | null>(null);
  const [formError, setFormError] = useState('');

  const [docFormOpened, docFormHandlers] = useDisclosure(false);
  const [editingDoc, setEditingDoc] = useState<Document | null>(null);
  const [deletingDocId, setDeletingDocId] = useState<string | null>(null);

  const people = peopleQuery.data?.items ?? [];
  const events = eventsQuery.data?.items ?? [];
  const allDocs = docsQuery.data?.items ?? [];

  const eventDocs = editingEvent
    ? allDocs.filter((d) => d.life_event_id === editingEvent.id)
    : [];

  const createMutation = useMutation({
    mutationFn: ({ personId, fields }: { personId: string; fields: CreateLifeEventInput }) =>
      createLifeEvent(caseId!, { ...fields, person_id: personId }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['life-events', caseId] });
      modalHandlers.close();
    },
    onError: () => setFormError('Failed to save. Please try again.'),
  });

  const updateMutation = useMutation({
    mutationFn: async ({
      eventId,
      personId,
      originalPersonId,
      fields,
    }: {
      eventId: string;
      personId: string;
      originalPersonId: string;
      fields: UpdateLifeEventInput;
    }) => {
      const ops: Promise<unknown>[] = [updateLifeEvent(caseId!, eventId, fields)];
      if (personId !== originalPersonId) {
        ops.push(reassignLifeEvent(caseId!, eventId, personId));
      }
      await Promise.all(ops);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['life-events', caseId] });
      queryClient.invalidateQueries({ queryKey: ['cases', caseId] });
      modalHandlers.close();
    },
    onError: () => setFormError('Failed to save. Please try again.'),
  });

  const deleteMutation = useMutation({
    mutationFn: (eventId: string) => deleteLifeEvent(caseId!, eventId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['life-events', caseId] });
      queryClient.invalidateQueries({ queryKey: ['cases', caseId] });
      deleteHandlers.close();
      setDeletingEvent(null);
    },
    onError: () => setFormError('Failed to delete. Please try again.'),
  });

  const docDeleteMutation = useMutation({
    mutationFn: (docId: string) => deleteDocument(caseId!, docId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['documents', caseId] });
      queryClient.invalidateQueries({ queryKey: ['life-events', caseId] });
      setDeletingDocId(null);
    },
  });

  function openCreate() {
    setEditingEvent(null);
    setFormError('');
    setDeletingDocId(null);
    modalHandlers.open();
  }

  function openEdit(e: LifeEvent) {
    setEditingEvent(e);
    setFormError('');
    setDeletingDocId(null);
    modalHandlers.open();
  }

  function openAddDoc() { setEditingDoc(null); docFormHandlers.open(); }
  function openEditDoc(doc: Document) { setEditingDoc(doc); docFormHandlers.open(); }
  function handleDeleteDoc(doc: Document) {
    if (deletingDocId === doc.id) { docDeleteMutation.mutate(doc.id); }
    else { setDeletingDocId(doc.id); }
  }

  function openDelete(e: LifeEvent) {
    setDeletingEvent(e);
    setFormError('');
    deleteHandlers.open();
  }

  function handleSubmit(personId: string, fields: CreateLifeEventInput | UpdateLifeEventInput) {
    if (!personId) {
      setFormError('Person is required.');
      return;
    }
    if (!(fields as CreateLifeEventInput).event_type && !('event_type' in fields && fields.event_type)) {
      setFormError('Event type is required.');
      return;
    }
    setFormError('');
    if (editingEvent) {
      updateMutation.mutate({
        eventId: editingEvent.id,
        personId,
        originalPersonId: editingEvent.person_id,
        fields: fields as UpdateLifeEventInput,
      });
    } else {
      createMutation.mutate({ personId, fields: fields as CreateLifeEventInput });
    }
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;
  const isLoading = eventsQuery.isLoading || peopleQuery.isLoading;
  const isError = eventsQuery.isError || peopleQuery.isError;

  function modalTitle() {
    if (!editingEvent) return 'Add Life Event';
    return `${personName(editingEvent.person_id, people)} — ${eventTypeLabel(editingEvent.event_type)}`;
  }

  return (
    <>
      <Group justify="space-between" mb="md">
        <Title order={2}>Life Events</Title>
        <Button onClick={openCreate} disabled={people.length === 0}>Add Life Event</Button>
      </Group>

      {people.length === 0 && !isLoading && (
        <Alert color="blue" mb="md">Add people to this case before creating life events.</Alert>
      )}

      {isLoading && <Loader />}
      {isError && <Alert color="red">Failed to load data.</Alert>}

      {!isLoading && !isError && (
        <Table striped highlightOnHover withTableBorder>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Person</Table.Th>
              <Table.Th>Event</Table.Th>
              <Table.Th>Date</Table.Th>
              <Table.Th>Place</Table.Th>
              <Table.Th>Docs</Table.Th>
              <Table.Th />
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {events.length === 0 && (
              <Table.Tr>
                <Table.Td colSpan={6}>
                  <Text c="dimmed">No life events yet.</Text>
                </Table.Td>
              </Table.Tr>
            )}
            {events.map((e) => (
              <Table.Tr key={e.id} onClick={() => openEdit(e)} style={{ cursor: 'pointer' }}>
                <Table.Td>{personName(e.person_id, people)}</Table.Td>
                <Table.Td>{eventTypeLabel(e.event_type)}</Table.Td>
                <Table.Td>{e.event_date ?? '—'}</Table.Td>
                <Table.Td>{e.event_place ?? '—'}</Table.Td>
                <Table.Td>
                  {!e.has_documents && (
                    <Badge color="orange" variant="dot" size="sm">No docs</Badge>
                  )}
                </Table.Td>
                <Table.Td>
                  <Group gap="xs" justify="flex-end" onClick={(ev) => ev.stopPropagation()}>
                    <Button size="xs" variant="subtle" color="red" onClick={() => openDelete(e)}>
                      Delete
                    </Button>
                  </Group>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      )}

      <Modal
        opened={modalOpened}
        onClose={modalHandlers.close}
        title={modalTitle()}
        size="lg"
      >
        <Stack>
          <LifeEventForm
            people={people}
            editing={editingEvent}
            onSubmit={handleSubmit}
            onClose={modalHandlers.close}
            loading={isSaving}
            error={formError}
          />
          {editingEvent && (
            <>
              <Divider mt="xs" />
              <Group justify="space-between" align="center">
                <Text fw={500} size="sm">Documents</Text>
                <Button size="xs" variant="light" onClick={openAddDoc}>Add</Button>
              </Group>
              {eventDocs.length === 0 && (
                <Text size="sm" c="dimmed">No documents linked to this event yet.</Text>
              )}
              <Stack gap="xs">
                {eventDocs.map((doc) => (
                  <Group key={doc.id} justify="space-between" wrap="nowrap"
                    style={{ borderBottom: '1px solid var(--mantine-color-gray-2)', paddingBottom: 8 }}
                  >
                    <div>
                      <Text size="sm" fw={500}>{doc.title}</Text>
                      <Text size="xs" c="dimmed">
                        {doc.document_type.replace(/_/g, ' ')} · {doc.official_copy_status?.label ?? '—'}
                      </Text>
                    </div>
                    {deletingDocId === doc.id ? (
                      <Group gap="xs" wrap="nowrap">
                        <Text size="xs" c="dimmed">Delete?</Text>
                        <Button size="xs" variant="subtle" onClick={() => setDeletingDocId(null)}>Cancel</Button>
                        <Button size="xs" color="red" onClick={() => handleDeleteDoc(doc)}>Delete</Button>
                      </Group>
                    ) : (
                      <Group gap="xs" wrap="nowrap">
                        <Button size="xs" variant="subtle" onClick={() => openEditDoc(doc)}>Edit</Button>
                        <Button size="xs" variant="subtle" color="red" onClick={() => handleDeleteDoc(doc)}>Delete</Button>
                      </Group>
                    )}
                  </Group>
                ))}
              </Stack>
            </>
          )}
        </Stack>
      </Modal>

      <DocumentFormModal
        caseId={caseId!}
        editing={editingDoc}
        opened={docFormOpened}
        onClose={docFormHandlers.close}
        onSuccess={docFormHandlers.close}
        lockedPersonId={editingEvent?.person_id}
        lockedLifeEventId={editingEvent?.id}
      />

      <Modal opened={deleteOpened} onClose={deleteHandlers.close} title="Delete Life Event">
        <Stack>
          <Text>
            Move this{' '}
            <strong>
              {deletingEvent ? eventTypeLabel(deletingEvent.event_type) : ''} event
            </strong>{' '}
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
              onClick={() => deletingEvent && deleteMutation.mutate(deletingEvent.id)}
            >
              Move to Trash
            </Button>
          </Group>
        </Stack>
      </Modal>
    </>
  );
}
