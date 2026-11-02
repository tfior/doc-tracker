/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Title, Text, Loader, Stack } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { restoreCase, permanentDeleteCase, type TrashedCase } from '../../api/trash';
import { globalTrashQuery } from '../../api/queries';
import ErrorMessage from '../../components/ErrorMessage';
import TrashRow from './TrashRow';
import PermanentDeleteModal, { type PermanentDeleteTarget } from './PermanentDeleteModal';

export default function GlobalTrashPage() {
  const queryClient = useQueryClient();
  const trashQuery = useQuery(globalTrashQuery());

  const [deleteOpened, deleteHandlers] = useDisclosure(false);
  const [pendingDelete, setPendingDelete] = useState<PermanentDeleteTarget | null>(null);

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['global-trash'] });
    queryClient.invalidateQueries({ queryKey: ['cases'] });
  };

  const restore = useMutation({
    mutationFn: (c: TrashedCase) => restoreCase(c.id),
    onSuccess: invalidate,
  });

  function openPermanentDelete(c: TrashedCase) {
    restore.reset();
    setPendingDelete({
      key: `case/${c.id}`,
      label: `case ${c.title}`,
      goesWith: 'All of its people, life events, documents and claim lines are permanently deleted too.',
      run: () => permanentDeleteCase(c.id),
    });
    deleteHandlers.open();
  }

  if (trashQuery.isLoading) return <Loader />;
  if (trashQuery.error) return <ErrorMessage error={trashQuery.error} title="Could not load the trash" />;

  const cases = trashQuery.data!.cases;

  return (
    <>
      <Title order={2} mb="md">Trash</Title>
      <Text c="dimmed" size="sm" mb="lg">
        Deleted cases appear here. Entities deleted within an active case (people, life events,
        documents, claim lines) are accessible from that case's Trash tab.
      </Text>

      <ErrorMessage error={restore.error} title={restore.variables && `Could not restore ${restore.variables.title}`} />

      {cases.length === 0 && <Text c="dimmed">No deleted cases.</Text>}

      <Stack gap="xs" mt="md">
        {cases.map((c) => (
          <TrashRow
            key={c.id}
            label={c.title}
            deletedAt={c.deleted_at}
            restoring={restore.isPending && restore.variables?.id === c.id}
            onRestore={() => restore.mutate(c)}
            onDelete={() => openPermanentDelete(c)}
          />
        ))}
      </Stack>

      <PermanentDeleteModal
        target={pendingDelete}
        opened={deleteOpened}
        onClose={deleteHandlers.close}
        onDeleted={invalidate}
      />
    </>
  );
}
