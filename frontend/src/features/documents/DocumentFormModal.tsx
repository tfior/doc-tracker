import { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Modal, Stack, Group, TextInput, Select, Checkbox, Divider,
  Button, Alert,
} from '@mantine/core';
import { listPeople } from '../../api/people';
import { listLifeEvents, type LifeEvent } from '../../api/lifeevents';
import {
  listDocumentStatuses, createDocument, updateDocument,
  transitionStatus, reassignDocument,
  type Document, type DocumentStatus,
} from '../../api/documents';

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
type Phase = typeof PHASES[number];

const PHASE_LABELS: Record<Phase, string> = {
  official_copy: 'Official Copy',
  amendment: 'Amendment',
  apostille: 'Apostille',
  translation: 'Translation',
};

// ---------------------------------------------------------------------------
// Form state
// ---------------------------------------------------------------------------

interface FormState {
  title: string;
  documentType: string;
  personId: string;
  lifeEventId: string;
  issuingAuthority: string;
  issueDate: string;
  recordedDate: string;
  recordedGivenName: string;
  recordedSurname: string;
  recordedBirthDate: string;
  recordedBirthPlace: string;
  isVerified: boolean;
  notes: string;
  officialCopyStatusId: string;
  amendmentStatusId: string;
  apostilleStatusId: string;
  translationStatusId: string;
}

function initForm(
  doc: Document | null,
  lockedPersonId?: string,
  lockedLifeEventId?: string,
): FormState {
  return {
    title: doc?.title ?? '',
    documentType: doc?.document_type ?? '',
    personId: doc?.person_id ?? lockedPersonId ?? '',
    lifeEventId: doc?.life_event_id ?? lockedLifeEventId ?? '',
    issuingAuthority: doc?.issuing_authority ?? '',
    issueDate: doc?.issue_date ?? '',
    recordedDate: doc?.recorded_date ?? '',
    recordedGivenName: doc?.recorded_given_name ?? '',
    recordedSurname: doc?.recorded_surname ?? '',
    recordedBirthDate: doc?.recorded_birth_date ?? '',
    recordedBirthPlace: doc?.recorded_birth_place ?? '',
    isVerified: doc?.is_verified ?? false,
    notes: doc?.notes ?? '',
    officialCopyStatusId: doc?.official_copy_status?.id ?? '',
    amendmentStatusId: doc?.amendment_status?.id ?? '',
    apostilleStatusId: doc?.apostille_status?.id ?? '',
    translationStatusId: doc?.translation_status?.id ?? '',
  };
}

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface DocumentFormModalProps {
  caseId: string;
  editing: Document | null;
  opened: boolean;
  onClose: () => void;
  onSuccess: () => void;
  /** When set, the person field is hidden and this value is used as person_id. */
  lockedPersonId?: string;
  /** When set, the life event field is hidden and this value is used as life_event_id. */
  lockedLifeEventId?: string;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function DocumentFormModal({
  caseId, editing, opened, onClose, onSuccess,
  lockedPersonId, lockedLifeEventId,
}: DocumentFormModalProps) {
  const queryClient = useQueryClient();

  const peopleQuery = useQuery({
    queryKey: ['people', caseId],
    queryFn: () => listPeople(caseId),
    enabled: opened,
  });

  const eventsQuery = useQuery({
    queryKey: ['life-events', caseId],
    queryFn: () => listLifeEvents(caseId),
    enabled: opened,
  });

  const statusesQuery = useQuery({
    queryKey: ['document-statuses'],
    queryFn: listDocumentStatuses,
    staleTime: 5 * 60 * 1000,
    enabled: opened,
  });

  const [form, setForm] = useState<FormState>(() =>
    initForm(editing, lockedPersonId, lockedLifeEventId),
  );
  const [formError, setFormError] = useState('');

  // Reset form when modal opens with a different document
  useEffect(() => {
    if (opened) {
      setForm(initForm(editing, lockedPersonId, lockedLifeEventId));
      setFormError('');
    }
  }, [opened, editing?.id]);

  const people = peopleQuery.data?.items ?? [];
  const lifeEvents = eventsQuery.data?.items ?? [];
  const allStatuses: DocumentStatus[] = statusesQuery.data ?? [];

  function statusesForPhase(phase: string) {
    return allStatuses
      .filter((s) => s.phase === phase || s.phase === 'any')
      .map((s) => ({ value: s.id, label: s.label }));
  }

  function updateForm(key: keyof FormState, value: string | boolean) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  function handlePersonChange(personId: string) {
    setForm((prev) => {
      const lifeEventStillValid = lifeEvents
        .filter((le) => le.person_id === personId)
        .some((le) => le.id === prev.lifeEventId);
      return { ...prev, personId, lifeEventId: lifeEventStillValid ? prev.lifeEventId : '' };
    });
  }

  // ---------------------------------------------------------------------------
  // Mutations
  // ---------------------------------------------------------------------------

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['documents', caseId] });
    queryClient.invalidateQueries({ queryKey: ['life-events', caseId] });
  };

  const createMutation = useMutation({
    mutationFn: () =>
      createDocument(caseId, {
        person_id: form.personId,
        document_type: form.documentType,
        title: form.title,
        life_event_id: form.lifeEventId || null,
        issuing_authority: form.issuingAuthority || null,
        issue_date: form.issueDate || null,
        recorded_date: form.recordedDate || null,
        recorded_given_name: form.recordedGivenName || null,
        recorded_surname: form.recordedSurname || null,
        recorded_birth_date: form.recordedBirthDate || null,
        recorded_birth_place: form.recordedBirthPlace || null,
        notes: form.notes || null,
        is_verified: form.isVerified,
      }),
    onSuccess: () => { invalidate(); onSuccess(); },
    onError: () => setFormError('Failed to save. Please try again.'),
  });

  const updateMutation = useMutation({
    mutationFn: async () => {
      if (!editing) return;

      const ops: Promise<unknown>[] = [
        updateDocument(caseId, editing.id, {
          title: form.title,
          document_type: form.documentType,
          issuing_authority: form.issuingAuthority || null,
          issue_date: form.issueDate || null,
          recorded_date: form.recordedDate || null,
          recorded_given_name: form.recordedGivenName || null,
          recorded_surname: form.recordedSurname || null,
          recorded_birth_date: form.recordedBirthDate || null,
          recorded_birth_place: form.recordedBirthPlace || null,
          notes: form.notes || null,
          is_verified: form.isVerified,
        }),
      ];

      const personChanged = form.personId !== editing.person_id;
      const lifeEventChanged = form.lifeEventId !== (editing.life_event_id ?? '');
      if (personChanged || lifeEventChanged) {
        ops.push(
          reassignDocument(caseId, editing.id, {
            person_id: form.personId,
            life_event_id: form.lifeEventId || null,
          }),
        );
      }

      const phaseChecks: { phase: Phase; field: keyof FormState; current: string }[] = [
        { phase: 'official_copy', field: 'officialCopyStatusId', current: editing.official_copy_status.id },
        { phase: 'amendment',     field: 'amendmentStatusId',    current: editing.amendment_status.id },
        { phase: 'apostille',     field: 'apostilleStatusId',    current: editing.apostille_status.id },
        { phase: 'translation',   field: 'translationStatusId',  current: editing.translation_status.id },
      ];
      for (const { phase, field, current } of phaseChecks) {
        const newId = form[field] as string;
        if (newId && newId !== current) {
          ops.push(transitionStatus(caseId, editing.id, phase, newId));
        }
      }

      await Promise.all(ops);
    },
    onSuccess: () => { invalidate(); onSuccess(); },
    onError: () => setFormError('Failed to save. Please try again.'),
  });

  function handleSubmit() {
    if (!form.personId) { setFormError('Person is required.'); return; }
    if (!form.documentType) { setFormError('Document type is required.'); return; }
    if (!form.title.trim()) { setFormError('Title is required.'); return; }
    setFormError('');
    if (editing) {
      updateMutation.mutate();
    } else {
      createMutation.mutate();
    }
  }

  const isSaving = createMutation.isPending || updateMutation.isPending;

  // ---------------------------------------------------------------------------
  // Life event options — filtered to selected person (or locked person)
  // ---------------------------------------------------------------------------

  const effectivePersonId = lockedPersonId ?? form.personId;

  const lifeEventOptions: { value: string; label: string }[] = [
    { value: '', label: 'None (unlinked)' },
    ...lifeEvents
      .filter((le: LifeEvent) => le.person_id === effectivePersonId)
      .map((le: LifeEvent) => ({
        value: le.id,
        label: `${le.event_type.replace(/_/g, ' ')}${le.event_date ? ` — ${le.event_date}` : ''}`,
      })),
  ];

  const personOptions = people.map((p) => ({
    value: p.id,
    label: `${p.first_name} ${p.last_name}`,
  }));

  const title = editing ? `Edit — ${editing.title}` : 'Add Document';

  return (
    <Modal opened={opened} onClose={onClose} title={title} size="xl">
      <Stack>
        {/* Person — hidden when locked */}
        {!lockedPersonId && (
          <Select
            label="Person"
            data={personOptions}
            value={form.personId}
            onChange={(v) => handlePersonChange(v ?? '')}
            searchable
            required
          />
        )}
        <Group grow>
          <Select
            label="Type"
            data={DOC_TYPE_OPTIONS}
            value={form.documentType}
            onChange={(v) => updateForm('documentType', v ?? '')}
            required
          />
          <TextInput
            label="Title"
            value={form.title}
            onChange={(e) => updateForm('title', e.currentTarget.value)}
            required
          />
        </Group>

        {/* Life event — hidden when locked */}
        {!lockedLifeEventId && (
          <Select
            label="Life Event"
            description="Optional — links this document to a specific event"
            data={lifeEventOptions}
            value={form.lifeEventId}
            onChange={(v) => updateForm('lifeEventId', v ?? '')}
            disabled={!effectivePersonId}
          />
        )}

        <Divider label="Document details" labelPosition="left" />
        <Group grow>
          <TextInput
            label="Issuing Authority"
            value={form.issuingAuthority}
            onChange={(e) => updateForm('issuingAuthority', e.currentTarget.value)}
          />
          <TextInput
            label="Issue Date"
            placeholder="YYYY-MM-DD"
            value={form.issueDate}
            onChange={(e) => updateForm('issueDate', e.currentTarget.value)}
          />
        </Group>
        <TextInput
          label="Date Recorded"
          placeholder="YYYY-MM-DD"
          value={form.recordedDate}
          onChange={(e) => updateForm('recordedDate', e.currentTarget.value)}
        />

        <Divider label="Recorded information" labelPosition="left" />
        <Group grow>
          <TextInput
            label="Recorded Given Name"
            value={form.recordedGivenName}
            onChange={(e) => updateForm('recordedGivenName', e.currentTarget.value)}
          />
          <TextInput
            label="Recorded Surname"
            value={form.recordedSurname}
            onChange={(e) => updateForm('recordedSurname', e.currentTarget.value)}
          />
        </Group>
        <Group grow>
          <TextInput
            label="Recorded Birth Date"
            placeholder="YYYY-MM-DD"
            value={form.recordedBirthDate}
            onChange={(e) => updateForm('recordedBirthDate', e.currentTarget.value)}
          />
          <TextInput
            label="Recorded Birth Place"
            value={form.recordedBirthPlace}
            onChange={(e) => updateForm('recordedBirthPlace', e.currentTarget.value)}
          />
        </Group>

        {editing && (
          <>
            <Divider label="Collection status" labelPosition="left" />
            {PHASES.map((phase) => (
              <Select
                key={phase}
                label={PHASE_LABELS[phase]}
                data={statusesForPhase(phase)}
                value={form[`${phase}StatusId` as keyof FormState] as string}
                onChange={(v) => updateForm(`${phase}StatusId` as keyof FormState, v ?? '')}
              />
            ))}
          </>
        )}

        <Divider label="Notes & verification" labelPosition="left" />
        <TextInput
          label="Notes"
          value={form.notes}
          onChange={(e) => updateForm('notes', e.currentTarget.value)}
        />
        <Checkbox
          label="Verified"
          checked={form.isVerified}
          onChange={(e) => updateForm('isVerified', e.currentTarget.checked)}
        />

        {formError && <Alert color="red">{formError}</Alert>}
        <Group justify="flex-end">
          <Button variant="default" onClick={onClose} disabled={isSaving}>Cancel</Button>
          <Button onClick={handleSubmit} loading={isSaving}>
            {editing ? 'Save' : 'Add Document'}
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}
