import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import {
  Title, Text, Loader, Alert, Group, Badge, Stack,
  Card, List, ThemeIcon, Button, Modal, TextInput, Select,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { getCase, updateCase, type Case } from '../../api/cases';
import { listLifeEvents } from '../../api/lifeevents';

const CLAIM_STATUS_COLOR: Record<string, string> = {
  not_yet_researched: 'gray',
  researching: 'blue',
  paused: 'orange',
  ineligible: 'red',
  eligible: 'green',
};

const STATUS_OPTIONS = [
  { value: 'active', label: 'Active' },
  { value: 'archived', label: 'Archived' },
  { value: 'complete', label: 'Complete' },
];

const CLAIM_STATUS_LABELS: Record<string, string> = {
  not_yet_researched: 'Not Yet Researched',
  researching: 'Researching',
  paused: 'Paused',
  ineligible: 'Ineligible',
  eligible: 'Eligible',
};

function formatEventType(type: string): string {
  return type.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

export default function CaseOverviewPage() {
  const { caseId } = useParams<{ caseId: string }>();
  const queryClient = useQueryClient();

  const caseQuery = useQuery({
    queryKey: ['cases', caseId],
    queryFn: () => getCase(caseId!),
  });

  const eventsQuery = useQuery({
    queryKey: ['life-events', caseId],
    queryFn: () => listLifeEvents(caseId!),
  });

  const [editOpened, editHandlers] = useDisclosure(false);
  const [title, setTitle] = useState('');
  const [status, setStatus] = useState('active');
  const [formError, setFormError] = useState('');

  const updateMutation = useMutation({
    mutationFn: (input: { title?: string; status?: string }) =>
      updateCase(caseId!, input),
    onSuccess: (updated: Case) => {
      queryClient.setQueryData(['cases', caseId], (old: typeof caseQuery.data) =>
        old ? { ...old, ...updated } : old,
      );
      queryClient.invalidateQueries({ queryKey: ['cases'] });
      editHandlers.close();
    },
    onError: () => setFormError('Failed to update case. Please try again.'),
  });

  function openEdit() {
    if (!caseQuery.data) return;
    setTitle(caseQuery.data.title);
    setStatus(caseQuery.data.status);
    setFormError('');
    editHandlers.open();
  }

  function handleSubmit() {
    if (!title.trim()) {
      setFormError('Title is required.');
      return;
    }
    setFormError('');
    updateMutation.mutate({ title: title.trim(), status });
  }

  if (caseQuery.isLoading) return <Loader />;
  if (caseQuery.isError) return <Alert color="red">Failed to load case.</Alert>;

  const detail = caseQuery.data!;
  const { claim_line_summary: cls } = detail;
  const flaggedEvents = eventsQuery.data?.items.filter((e) => !e.has_documents) ?? [];

  const claimStatuses = ['eligible', 'researching', 'not_yet_researched', 'paused', 'ineligible'] as const;

  return (
    <>
      <Stack gap="lg">
        <Group justify="space-between" align="flex-start">
          <div>
            <Title order={2}>{detail.title}</Title>
            <Text c="dimmed" size="sm" mt={4}>Case Overview</Text>
          </div>
          <Group gap="xs">
            <Badge
              size="lg"
              color={detail.status === 'active' ? 'green' : detail.status === 'complete' ? 'blue' : 'gray'}
            >
              {detail.status}
            </Badge>
            <Button size="xs" variant="default" onClick={openEdit}>Edit</Button>
          </Group>
        </Group>

        {/* Claim Line Summary */}
        <Card withBorder padding="lg" maw={400}>
          <Title order={4} mb="md">Claim Lines</Title>
          <Stack gap={6}>
            {claimStatuses.map((s) =>
              cls[s] > 0 ? (
                <Group key={s} gap="xs">
                  <Badge color={CLAIM_STATUS_COLOR[s]} variant="dot" size="sm">
                    {CLAIM_STATUS_LABELS[s]}
                  </Badge>
                  <Text size="sm">{cls[s]}</Text>
                </Group>
              ) : null,
            )}
            {cls.total === 0 && <Text size="sm" c="dimmed">No claim lines.</Text>}
          </Stack>
        </Card>

        {/* Life Events Without Documents */}
        {!eventsQuery.isLoading && flaggedEvents.length > 0 && (
          <Card withBorder padding="lg">
            <Title order={4} mb="xs">Life Events Missing Documents</Title>
            <Text size="sm" c="dimmed" mb="sm">
              These events have no documents associated yet.
            </Text>
            <List spacing="xs" size="sm">
              {flaggedEvents.map((e) => (
                <List.Item key={e.id}>
                  <Group gap="xs">
                    <ThemeIcon color="orange" size="xs" radius="xl" variant="filled">
                      <span />
                    </ThemeIcon>
                    <Text size="sm">
                      {formatEventType(e.event_type)}
                      {e.event_date ? ` — ${e.event_date}` : ''}
                    </Text>
                  </Group>
                </List.Item>
              ))}
            </List>
          </Card>
        )}

      </Stack>

      {/* Edit modal */}
      <Modal opened={editOpened} onClose={editHandlers.close} title="Edit Case">
        <Stack>
          <TextInput
            label="Title"
            value={title}
            onChange={(e) => setTitle(e.currentTarget.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSubmit()}
            required
          />
          <Select
            label="Status"
            data={STATUS_OPTIONS}
            value={status}
            onChange={(v) => setStatus(v ?? 'active')}
          />
          {formError && <Alert color="red">{formError}</Alert>}
          <Group justify="flex-end">
            <Button variant="default" onClick={editHandlers.close} disabled={updateMutation.isPending}>
              Cancel
            </Button>
            <Button onClick={handleSubmit} loading={updateMutation.isPending}>
              Save
            </Button>
          </Group>
        </Stack>
      </Modal>
    </>
  );
}
