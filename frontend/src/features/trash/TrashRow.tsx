/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { Group, Text, Button } from '@mantine/core';

interface TrashRowProps {
  label: string;
  meta?: string;
  deletedAt: string;
  restoring: boolean;
  onRestore: () => void;
  onDelete: () => void;
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString();
}

export default function TrashRow({ label, meta, deletedAt, restoring, onRestore, onDelete }: TrashRowProps) {
  return (
    <Group justify="space-between" wrap="nowrap" py="xs"
      style={{ borderBottom: '1px solid var(--mantine-color-gray-2)' }}
    >
      <div>
        <Text size="sm" fw={500}>{label}</Text>
        {meta && <Text size="xs" c="dimmed">{meta}</Text>}
        <Text size="xs" c="dimmed">Deleted {formatDate(deletedAt)}</Text>
      </div>
      <Group gap="xs" wrap="nowrap">
        <Button size="xs" variant="light" loading={restoring} onClick={onRestore}>
          Restore
        </Button>
        <Button size="xs" variant="subtle" color="red" onClick={onDelete}>
          Delete Permanently
        </Button>
      </Group>
    </Group>
  );
}
