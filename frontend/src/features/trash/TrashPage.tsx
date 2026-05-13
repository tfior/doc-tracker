import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import {
  Title, Text, Loader, Alert, Button, Group, Stack,
  Modal, Divider,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  getCaseTrash, restoreEntity, permanentDeleteEntity,
  type TrashEntityType,
} from '../../api/trash';

interface PendingDelete {
  type: TrashEntityType;
  id: string;
  label: string;
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString();
}

function eventTypeLabel(type: string) {
  return type.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

function docTypeLabel(type: string) {
  return type.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

function claimStatusLabel(status: string) {
  return status.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase());
}

export default function TrashPage() {
  const { caseId } = useParams<{ caseId: string }>();
  const queryClient = useQueryClient();

  const trashQuery = useQuery({
    queryKey: ['trash', caseId],
    queryFn: () => getCaseTrash(caseId!),
  });

  const [deleteOpened, deleteHandlers] = useDisclosure(false);
  const [pendingDelete, setPendingDelete] = useState<PendingDelete | null>(null);

  const invalidateAll = () => {
    queryClient.invalidateQueries({ queryKey: ['trash', caseId] });
    queryClient.invalidateQueries({ queryKey: ['people', caseId] });
    queryClient.invalidateQueries({ queryKey: ['life-events', caseId] });
    queryClient.invalidateQueries({ queryKey: ['documents', caseId] });
    queryClient.invalidateQueries({ queryKey: ['claim-lines', caseId] });
    queryClient.invalidateQueries({ queryKey: ['cases', caseId] });
  };

  const restoreMutation = useMutation({
    mutationFn: ({ type, id }: { type: TrashEntityType; id: string }) =>
      restoreEntity(caseId!, type, id),
    onSuccess: invalidateAll,
  });

  const permanentDeleteMutation = useMutation({
    mutationFn: ({ type, id }: { type: TrashEntityType; id: string }) =>
      permanentDeleteEntity(caseId!, type, id),
    onSuccess: () => { invalidateAll(); deleteHandlers.close(); setPendingDelete(null); },
  });

  function openPermanentDelete(type: TrashEntityType, id: string, label: string) {
    setPendingDelete({ type, id, label });
    deleteHandlers.open();
  }

  const trash = trashQuery.data;
  const isEmpty =
    trash &&
    trash.people.length === 0 &&
    trash.life_events.length === 0 &&
    trash.documents.length === 0 &&
    trash.claim_lines.length === 0;

  if (trashQuery.isLoading) return <Loader />;
  if (trashQuery.isError) return <Alert color="red">Failed to load trash.</Alert>;

  function TrashRow({
    label,
    meta,
    deletedAt,
    type,
    id,
    deleteLabel,
  }: {
    label: string;
    meta?: string;
    deletedAt: string;
    type: TrashEntityType;
    id: string;
    deleteLabel: string;
  }) {
    const isRestoring = restoreMutation.isPending &&
      restoreMutation.variables?.type === type &&
      restoreMutation.variables?.id === id;

    return (
      <Group justify="space-between" wrap="nowrap" py="xs"
        style={{ borderBottom: '1px solid var(--mantine-color-gray-2)' }}
      >
        <div>
          <Text size="sm" fw={500}>{label}</Text>
          {meta && <Text size="xs" c="dimmed">{meta}</Text>}
          <Text size="xs" c="dimmed">Deleted {formatDate(deletedAt)}</Text>
        </div>
        <Group gap="xs" wrap="nowrap">
          <Button
            size="xs"
            variant="light"
            loading={isRestoring}
            onClick={() => restoreMutation.mutate({ type, id })}
          >
            Restore
          </Button>
          <Button
            size="xs"
            variant="subtle"
            color="red"
            onClick={() => openPermanentDelete(type, id, deleteLabel)}
          >
            Delete Permanently
          </Button>
        </Group>
      </Group>
    );
  }

  return (
    <>
      <Title order={2} mb="md">Trash</Title>

      {isEmpty && (
        <Text c="dimmed">Trash is empty.</Text>
      )}

      <Stack gap="xl">
        {trash!.people.length > 0 && (
          <div>
            <Title order={4} mb="xs">People</Title>
            {trash!.people.map((p) => (
              <TrashRow
                key={p.id}
                label={`${p.first_name} ${p.last_name}`}
                deletedAt={p.deleted_at}
                type="people"
                id={p.id}
                deleteLabel={`${p.first_name} ${p.last_name}`}
              />
            ))}
          </div>
        )}

        {trash!.life_events.length > 0 && (
          <div>
            <Title order={4} mb="xs">Life Events</Title>
            {trash!.life_events.map((le) => (
              <TrashRow
                key={le.id}
                label={eventTypeLabel(le.event_type)}
                meta={le.event_date ?? undefined}
                deletedAt={le.deleted_at}
                type="life-events"
                id={le.id}
                deleteLabel={`${eventTypeLabel(le.event_type)} life event`}
              />
            ))}
          </div>
        )}

        {trash!.documents.length > 0 && (
          <div>
            <Title order={4} mb="xs">Documents</Title>
            {trash!.documents.map((d) => (
              <TrashRow
                key={d.id}
                label={d.title}
                meta={docTypeLabel(d.document_type)}
                deletedAt={d.deleted_at}
                type="documents"
                id={d.id}
                deleteLabel={d.title}
              />
            ))}
          </div>
        )}

        {trash!.claim_lines.length > 0 && (
          <div>
            <Title order={4} mb="xs">Claim Lines</Title>
            {trash!.claim_lines.map((cl) => (
              <TrashRow
                key={cl.id}
                label={`Claim Line — ${claimStatusLabel(cl.status)}`}
                deletedAt={cl.deleted_at}
                type="claim-lines"
                id={cl.id}
                deleteLabel="this claim line"
              />
            ))}
          </div>
        )}
      </Stack>

      <Modal
        opened={deleteOpened}
        onClose={deleteHandlers.close}
        title="Delete Permanently"
      >
        <Stack>
          <Text>
            Permanently delete <strong>{pendingDelete?.label}</strong>? This cannot be undone.
          </Text>
          <Divider />
          <Group justify="flex-end">
            <Button
              variant="default"
              onClick={deleteHandlers.close}
              disabled={permanentDeleteMutation.isPending}
            >
              Cancel
            </Button>
            <Button
              color="red"
              loading={permanentDeleteMutation.isPending}
              onClick={() =>
                pendingDelete &&
                permanentDeleteMutation.mutate({
                  type: pendingDelete.type,
                  id: pendingDelete.id,
                })
              }
            >
              Delete Permanently
            </Button>
          </Group>
        </Stack>
      </Modal>
    </>
  );
}
