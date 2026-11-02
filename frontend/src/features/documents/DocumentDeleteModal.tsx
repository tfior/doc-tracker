/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useQueryClient } from '@tanstack/react-query';
import { Text } from '@mantine/core';
import { deleteDocument, type Document } from '../../api/documents';
import { documentsQuery, invalidateCase } from '../../api/queries';
import ConfirmDeleteModal from '../../components/ConfirmDeleteModal';
import { useVersionedDelete } from '../../components/useVersionedDelete';

interface DocumentDeleteModalProps {
  caseId: string;
  record: Document | null;
  opened: boolean;
  onClose: () => void;
}

export default function DocumentDeleteModal({ caseId, record, opened, onClose }: DocumentDeleteModalProps) {
  const queryClient = useQueryClient();
  const d = useVersionedDelete({
    record, opened, onClose,
    remove: (doc) => deleteDocument(caseId, doc.id, doc.version),
    load: async () => (await queryClient.fetchQuery({ ...documentsQuery(caseId), staleTime: 0 })).items,
    onDeleted: () => invalidateCase(queryClient, caseId),
  });

  return (
    <ConfirmDeleteModal
      opened={opened}
      onClose={d.close}
      title="Delete Document"
      onConfirm={d.confirm}
      pending={d.pending}
      error={d.error}
      onReload={d.reload}
    >
      <Text>
        Move <strong>{d.current?.title}</strong> to the trash? Nothing else goes with it. It can be
        restored from the case's trash.
      </Text>
    </ConfirmDeleteModal>
  );
}
