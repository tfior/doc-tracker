/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Text, Loader } from '@mantine/core';
import { deletePerson, getPersonDeletionImpact, type Person } from '../../api/people';
import { invalidateCase, peopleQuery } from '../../api/queries';
import ConfirmDeleteModal from '../../components/ConfirmDeleteModal';
import ErrorMessage from '../../components/ErrorMessage';
import { useVersionedDelete } from '../../components/useVersionedDelete';
import { personImpactText } from './deletionImpactText';
import { fullName } from './personName';

interface PersonDeleteModalProps {
  caseId: string;
  record: Person | null;
  opened: boolean;
  onClose: () => void;
}

export default function PersonDeleteModal({ caseId, record, opened, onClose }: PersonDeleteModalProps) {
  const queryClient = useQueryClient();

  const impact = useQuery({
    queryKey: ['deletion-impact', 'person', record?.id],
    queryFn: () => getPersonDeletionImpact(caseId, record!.id),
    enabled: opened && !!record,
    staleTime: 0,
    gcTime: 0,
  });

  const d = useVersionedDelete({
    record, opened, onClose,
    remove: (p) => deletePerson(caseId, p.id, p.version),
    load: async () => (await queryClient.fetchQuery({ ...peopleQuery(caseId), staleTime: 0 })).items,
    onDeleted: () => invalidateCase(queryClient, caseId),
  });

  async function reload() {
    await Promise.all([d.reload(), impact.refetch()]);
  }

  return (
    <ConfirmDeleteModal
      opened={opened}
      onClose={d.close}
      title="Delete Person"
      onConfirm={d.confirm}
      canConfirm={impact.isSuccess}
      pending={d.pending}
      error={d.error}
      onReload={reload}
    >
      <Text>Move <strong>{d.current ? fullName(d.current) : ''}</strong> to the trash?</Text>
      {impact.isLoading && <Loader size="sm" />}
      <ErrorMessage error={impact.error} title="Could not work out what goes with this person" />
      {impact.isSuccess && (
        <Text>{personImpactText(impact.data, false)} Everything can be restored from the case's trash.</Text>
      )}
    </ConfirmDeleteModal>
  );
}
