/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useQueryClient } from '@tanstack/react-query';
import { Text } from '@mantine/core';
import { deleteCase, type Case } from '../../api/cases';
import { casesQuery } from '../../api/queries';
import ConfirmDeleteModal from '../../components/ConfirmDeleteModal';
import { useVersionedDelete } from '../../components/useVersionedDelete';

interface CaseDeleteModalProps {
  record: Case | null;
  opened: boolean;
  onClose: () => void;
}

export default function CaseDeleteModal({ record, opened, onClose }: CaseDeleteModalProps) {
  const queryClient = useQueryClient();
  const d = useVersionedDelete({
    record, opened, onClose,
    remove: (c) => deleteCase(c.id, c.version),
    load: async () => (await queryClient.fetchQuery({ ...casesQuery(), staleTime: 0 })).items,
    onDeleted: () => {
      queryClient.invalidateQueries({ queryKey: ['cases'] });
      queryClient.invalidateQueries({ queryKey: ['global-trash'] });
    },
  });

  return (
    <ConfirmDeleteModal
      opened={opened}
      onClose={d.close}
      title="Delete Case"
      onConfirm={d.confirm}
      pending={d.pending}
      error={d.error}
      onReload={d.reload}
    >
      <Text>
        Move <strong>{d.current?.title}</strong> to the trash? Everything in it goes with it: its
        people, life events, documents and claim lines. It can be restored from{' '}
        <strong>Trash</strong> in the sidebar.
      </Text>
    </ConfirmDeleteModal>
  );
}
