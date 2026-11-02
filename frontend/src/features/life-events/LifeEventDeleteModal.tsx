/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Text, Loader } from '@mantine/core';
import { deleteLifeEvent, getLifeEventDeletionImpact, type LifeEvent } from '../../api/lifeevents';
import { invalidateCase, lifeEventsQuery, peopleQuery } from '../../api/queries';
import ConfirmDeleteModal from '../../components/ConfirmDeleteModal';
import ErrorMessage from '../../components/ErrorMessage';
import { useVersionedDelete } from '../../components/useVersionedDelete';
import { personName } from '../people/personName';
import { lifeEventImpactText } from './deletionImpactText';
import { eventTypeLabel } from './options';

interface LifeEventDeleteModalProps {
  caseId: string;
  record: LifeEvent | null;
  opened: boolean;
  onClose: () => void;
}

export default function LifeEventDeleteModal({ caseId, record, opened, onClose }: LifeEventDeleteModalProps) {
  const queryClient = useQueryClient();
  const people = useQuery(peopleQuery(caseId)).data?.items ?? [];

  const impact = useQuery({
    queryKey: ['deletion-impact', 'life-event', record?.id],
    queryFn: () => getLifeEventDeletionImpact(caseId, record!.id),
    enabled: opened && !!record,
    staleTime: 0,
    gcTime: 0,
  });

  const d = useVersionedDelete({
    record, opened, onClose,
    remove: (le) => deleteLifeEvent(caseId, le.id, le.version),
    load: async () => (await queryClient.fetchQuery({ ...lifeEventsQuery(caseId), staleTime: 0 })).items,
    onDeleted: () => invalidateCase(queryClient, caseId),
  });

  async function reload() {
    await Promise.all([d.reload(), impact.refetch()]);
  }

  return (
    <ConfirmDeleteModal
      opened={opened}
      onClose={d.close}
      title="Delete Life Event"
      onConfirm={d.confirm}
      canConfirm={impact.isSuccess}
      pending={d.pending}
      error={d.error}
      onReload={reload}
    >
      {d.current && (
        <Text>
          Move the <strong>{eventTypeLabel(d.current.event_type)}</strong> event of{' '}
          <strong>{personName(d.current.person_id, people)}</strong> to the trash?
        </Text>
      )}
      {impact.isLoading && <Loader size="sm" />}
      <ErrorMessage error={impact.error} title="Could not work out what goes with this event" />
      {impact.isSuccess && (
        <Text>{lifeEventImpactText(impact.data, false)} Everything can be restored from the case's trash.</Text>
      )}
    </ConfirmDeleteModal>
  );
}
