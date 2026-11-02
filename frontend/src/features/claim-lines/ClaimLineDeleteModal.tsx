/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Text } from '@mantine/core';
import { deleteClaimLine, type ClaimLine } from '../../api/claimlines';
import { claimLinesQuery, invalidateCase, peopleQuery } from '../../api/queries';
import ConfirmDeleteModal from '../../components/ConfirmDeleteModal';
import { useVersionedDelete } from '../../components/useVersionedDelete';
import { personName } from '../people/personName';

interface ClaimLineDeleteModalProps {
  caseId: string;
  record: ClaimLine | null;
  opened: boolean;
  onClose: () => void;
}

export default function ClaimLineDeleteModal({ caseId, record, opened, onClose }: ClaimLineDeleteModalProps) {
  const queryClient = useQueryClient();
  const people = useQuery(peopleQuery(caseId)).data?.items ?? [];
  const d = useVersionedDelete({
    record, opened, onClose,
    remove: (line) => deleteClaimLine(caseId, line.id, line.version),
    load: async () => (await queryClient.fetchQuery({ ...claimLinesQuery(caseId), staleTime: 0 })).items,
    onDeleted: () => invalidateCase(queryClient, caseId),
  });

  return (
    <ConfirmDeleteModal
      opened={opened}
      onClose={d.close}
      title="Delete Claim Line"
      onConfirm={d.confirm}
      pending={d.pending}
      error={d.error}
      onReload={d.reload}
    >
      <Text>
        Move the claim line rooted at{' '}
        <strong>{d.current ? personName(d.current.root_person_id, people) : ''}</strong> to the trash?
        Nothing else goes with it. It can be restored from the case's trash.
      </Text>
    </ConfirmDeleteModal>
  );
}
