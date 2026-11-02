/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams } from 'react-router-dom';
import { Title, Text, Loader, Stack } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { restoreEntity, permanentDeleteEntity, type TrashEntityType } from '../../api/trash';
import { getPersonDeletionImpact } from '../../api/people';
import { getLifeEventDeletionImpact } from '../../api/lifeevents';
import { caseTrashQuery, invalidateCase } from '../../api/queries';
import ErrorMessage from '../../components/ErrorMessage';
import { eventTypeLabel } from '../life-events/options';
import { docTypeLabel } from '../documents/options';
import { claimStatusLabel } from '../claim-lines/options';
import { personImpactText } from '../people/deletionImpactText';
import { lifeEventImpactText } from '../life-events/deletionImpactText';
import TrashRow from './TrashRow';
import PermanentDeleteModal, { type PermanentDeleteTarget } from './PermanentDeleteModal';

interface TrashItem {
  type: TrashEntityType;
  id: string;
  label: string;
  meta?: string;
  deletedAt: string;
  loadGoesWith?: () => Promise<string>;
}

interface Restore {
  type: TrashEntityType;
  id: string;
  label: string;
}

export default function TrashPage() {
  const { caseId } = useParams<{ caseId: string }>();
  const queryClient = useQueryClient();
  const trashQuery = useQuery(caseTrashQuery(caseId!));

  const [deleteOpened, deleteHandlers] = useDisclosure(false);
  const [pendingDelete, setPendingDelete] = useState<PermanentDeleteTarget | null>(null);

  const restore = useMutation({
    mutationFn: ({ type, id }: Restore) => restoreEntity(caseId!, type, id),
    onSuccess: () => invalidateCase(queryClient, caseId!),
  });

  function openPermanentDelete(item: TrashItem) {
    restore.reset();
    setPendingDelete({
      key: `${item.type}/${item.id}`,
      label: item.label,
      loadGoesWith: item.loadGoesWith,
      run: () => permanentDeleteEntity(caseId!, item.type, item.id),
    });
    deleteHandlers.open();
  }

  if (trashQuery.isLoading) return <Loader />;
  if (trashQuery.error) return <ErrorMessage error={trashQuery.error} title="Could not load the trash" />;

  const trash = trashQuery.data!;
  const sections: { title: string; items: TrashItem[] }[] = [
    {
      title: 'People',
      items: trash.people.map((p) => ({
        type: 'people', id: p.id, label: `${p.first_name} ${p.last_name}`, deletedAt: p.deleted_at,
        loadGoesWith: async () => personImpactText(await getPersonDeletionImpact(caseId!, p.id), true),
      })),
    },
    {
      title: 'Life Events',
      items: trash.life_events.map((le) => ({
        type: 'life-events', id: le.id, label: `${eventTypeLabel(le.event_type)} life event`,
        meta: le.event_date ?? undefined, deletedAt: le.deleted_at,
        loadGoesWith: async () => lifeEventImpactText(await getLifeEventDeletionImpact(caseId!, le.id), true),
      })),
    },
    {
      title: 'Documents',
      items: trash.documents.map((d) => ({
        type: 'documents', id: d.id, label: d.title, meta: docTypeLabel(d.document_type), deletedAt: d.deleted_at,
      })),
    },
    {
      title: 'Claim Lines',
      items: trash.claim_lines.map((cl) => ({
        type: 'claim-lines', id: cl.id, label: `Claim line — ${claimStatusLabel(cl.status)}`, deletedAt: cl.deleted_at,
      })),
    },
  ];
  const isEmpty = sections.every((s) => s.items.length === 0);
  const failed = restore.variables;

  return (
    <>
      <Title order={2} mb="md">Trash</Title>
      <ErrorMessage error={restore.error} title={failed && `Could not restore ${failed.label}`} />

      {isEmpty && <Text c="dimmed">Trash is empty.</Text>}

      <Stack gap="xl" mt="md">
        {sections.filter((s) => s.items.length > 0).map((s) => (
          <div key={s.title}>
            <Title order={4} mb="xs">{s.title}</Title>
            {s.items.map((item) => (
              <TrashRow
                key={item.id}
                label={item.label}
                meta={item.meta}
                deletedAt={item.deletedAt}
                restoring={restore.isPending && restore.variables?.id === item.id}
                onRestore={() => restore.mutate({ type: item.type, id: item.id, label: item.label })}
                onDelete={() => openPermanentDelete(item)}
              />
            ))}
          </div>
        ))}
      </Stack>

      <PermanentDeleteModal
        target={pendingDelete}
        opened={deleteOpened}
        onClose={deleteHandlers.close}
        onDeleted={() => invalidateCase(queryClient, caseId!)}
      />
    </>
  );
}
