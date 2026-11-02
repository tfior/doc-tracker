/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import type { ReactNode } from 'react';
import { Modal, Stack, Group, Button } from '@mantine/core';
import ErrorMessage from './ErrorMessage';

interface ConfirmDeleteModalProps {
  opened: boolean;
  onClose: () => void;
  title: string;
  /** What is being deleted and what goes with it. */
  children: ReactNode;
  confirmLabel?: string;
  onConfirm: () => void;
  /** False while what goes with the item is still being worked out. */
  canConfirm?: boolean;
  pending: boolean;
  error: unknown;
  onReload?: () => Promise<void>;
}

export default function ConfirmDeleteModal({
  opened, onClose, title, children, confirmLabel = 'Move to Trash',
  onConfirm, canConfirm = true, pending, error, onReload,
}: ConfirmDeleteModalProps) {
  return (
    <Modal opened={opened} onClose={onClose} title={title}>
      <Stack>
        {children}
        <ErrorMessage error={error} onReload={onReload} />
        <Group justify="flex-end">
          <Button variant="default" onClick={onClose} disabled={pending}>Cancel</Button>
          <Button color="red" loading={pending} disabled={!canConfirm} onClick={onConfirm}>
            {confirmLabel}
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}
