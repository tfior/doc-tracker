import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  Title, Text, Loader, Alert, Button, Group, Stack, Modal, Divider,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import {
  getGlobalTrash, restoreCase, permanentDeleteCase,
  type TrashedCase,
} from '../../api/trash';

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString();
}

export default function GlobalTrashPage() {
  const queryClient = useQueryClient();

  const trashQuery = useQuery({
    queryKey: ['global-trash'],
    queryFn: getGlobalTrash,
  });

  const [deleteOpened, deleteHandlers] = useDisclosure(false);
  const [pendingCase, setPendingCase] = useState<TrashedCase | null>(null);

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['global-trash'] });
    queryClient.invalidateQueries({ queryKey: ['cases'] });
  };

  const restoreMutation = useMutation({
    mutationFn: (caseId: string) => restoreCase(caseId),
    onSuccess: invalidate,
  });

  const permanentDeleteMutation = useMutation({
    mutationFn: (caseId: string) => permanentDeleteCase(caseId),
    onSuccess: () => { invalidate(); deleteHandlers.close(); setPendingCase(null); },
  });

  const cases = trashQuery.data?.cases ?? [];

  if (trashQuery.isLoading) return <Loader />;
  if (trashQuery.isError) return <Alert color="red">Failed to load trash.</Alert>;

  return (
    <>
      <Title order={2} mb="md">Trash</Title>
      <Text c="dimmed" size="sm" mb="lg">
        Deleted cases appear here. Entities deleted within an active case (people, life events,
        documents, claim lines) are accessible from that case's Trash tab.
      </Text>

      {cases.length === 0 && <Text c="dimmed">No deleted cases.</Text>}

      <Stack gap="xs">
        {cases.map((c) => {
          const isRestoring =
            restoreMutation.isPending && restoreMutation.variables === c.id;
          return (
            <Group
              key={c.id}
              justify="space-between"
              wrap="nowrap"
              py="xs"
              style={{ borderBottom: '1px solid var(--mantine-color-gray-2)' }}
            >
              <div>
                <Text size="sm" fw={500}>{c.title}</Text>
                <Text size="xs" c="dimmed">Deleted {formatDate(c.deleted_at)}</Text>
              </div>
              <Group gap="xs" wrap="nowrap">
                <Button
                  size="xs"
                  variant="light"
                  loading={isRestoring}
                  onClick={() => restoreMutation.mutate(c.id)}
                >
                  Restore
                </Button>
                <Button
                  size="xs"
                  variant="subtle"
                  color="red"
                  onClick={() => { setPendingCase(c); deleteHandlers.open(); }}
                >
                  Delete Permanently
                </Button>
              </Group>
            </Group>
          );
        })}
      </Stack>

      <Modal opened={deleteOpened} onClose={deleteHandlers.close} title="Delete Permanently">
        <Stack>
          <Text>
            Permanently delete case <strong>{pendingCase?.title}</strong>? This will also permanently
            delete all people, life events, documents, and claim lines within it. This cannot be
            undone.
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
              onClick={() => pendingCase && permanentDeleteMutation.mutate(pendingCase.id)}
            >
              Delete Permanently
            </Button>
          </Group>
        </Stack>
      </Modal>
    </>
  );
}
