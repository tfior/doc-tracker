/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useMutation, useQuery } from '@tanstack/react-query';
import { Text, Loader } from '@mantine/core';
import ConfirmDeleteModal from '../../components/ConfirmDeleteModal';
import ErrorMessage from '../../components/ErrorMessage';

export interface PermanentDeleteTarget {
  /** Identifies the record, for caching what goes with it. */
  key: string;
  label: string;
  /** What is permanently deleted along with it, when that is known without asking the server. */
  goesWith?: string;
  /** Asks the server what is permanently deleted along with it. */
  loadGoesWith?: () => Promise<string>;
  run: () => Promise<void>;
}

interface PermanentDeleteModalProps {
  target: PermanentDeleteTarget | null;
  opened: boolean;
  onClose: () => void;
  onDeleted: () => void;
}

export default function PermanentDeleteModal({ target, opened, onClose, onDeleted }: PermanentDeleteModalProps) {
  const impact = useQuery({
    queryKey: ['deletion-impact', 'permanent', target?.key],
    queryFn: () => target!.loadGoesWith!(),
    enabled: opened && !!target?.loadGoesWith,
    staleTime: 0,
    gcTime: 0,
  });

  const del = useMutation({
    mutationFn: (t: PermanentDeleteTarget) => t.run(),
    onSuccess: () => {
      onDeleted();
      close();
    },
  });

  function close() {
    del.reset();
    onClose();
  }

  const goesWith = target?.loadGoesWith ? impact.data : target?.goesWith;

  return (
    <ConfirmDeleteModal
      opened={opened}
      onClose={close}
      title="Delete Permanently"
      confirmLabel="Delete Permanently"
      onConfirm={() => target && del.mutate(target)}
      canConfirm={!target?.loadGoesWith || impact.isSuccess}
      pending={del.isPending}
      error={del.error}
    >
      <Text>Permanently delete <strong>{target?.label}</strong>? This cannot be undone.</Text>
      {impact.isLoading && <Loader size="sm" />}
      <ErrorMessage error={impact.error} title="Could not work out what goes with it" />
      {goesWith && <Text>{goesWith}</Text>}
    </ConfirmDeleteModal>
  );
}
