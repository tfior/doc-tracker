/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/. */

import { useState } from 'react';
import { Divider, Group, Text, Button, Stack } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import type { LifeEvent } from '../../api/lifeevents';
import { eventTypeLabel } from './options';
import LifeEventFormModal from './LifeEventFormModal';
import LifeEventDeleteModal from './LifeEventDeleteModal';
import { formatQualifiedDate } from '../../lib/dates';

interface LifeEventListSectionProps {
  caseId: string;
  personId: string;
  lifeEvents: LifeEvent[];
}

/** A person's life events, managed from inside the person's modal. */
export default function LifeEventListSection({ caseId, personId, lifeEvents }: LifeEventListSectionProps) {
  const [formOpened, formHandlers] = useDisclosure(false);
  const [deleteOpened, deleteHandlers] = useDisclosure(false);
  const [editing, setEditing] = useState<LifeEvent | null>(null);
  const [deleting, setDeleting] = useState<LifeEvent | null>(null);

  function openForm(le: LifeEvent | null) {
    setEditing(le);
    formHandlers.open();
  }

  function openDelete(le: LifeEvent) {
    setDeleting(le);
    deleteHandlers.open();
  }

  return (
    <>
      <Divider mt="xs" />
      <Group justify="space-between" align="center">
        <Text fw={500} size="sm">Life Events</Text>
        <Button size="xs" variant="light" onClick={() => openForm(null)}>Add</Button>
      </Group>
      {lifeEvents.length === 0 && <Text size="sm" c="dimmed">No life events yet.</Text>}
      <Stack gap="xs">
        {lifeEvents.map((le) => (
          <Group key={le.id} justify="space-between" wrap="nowrap"
            style={{ borderBottom: '1px solid var(--mantine-color-gray-2)', paddingBottom: 8 }}
          >
            <div>
              <Text size="sm" fw={500}>{eventTypeLabel(le.event_type)}</Text>
              <Text size="xs" c="dimmed">
                {[le.event_date && formatQualifiedDate(le.event_date), le.event_place].filter(Boolean).join(' · ') || 'No date or place'}
              </Text>
            </div>
            <Group gap="xs" wrap="nowrap">
              <Button size="xs" variant="subtle" onClick={() => openForm(le)}>Edit</Button>
              <Button size="xs" variant="subtle" color="red" onClick={() => openDelete(le)}>Delete</Button>
            </Group>
          </Group>
        ))}
      </Stack>

      <LifeEventFormModal
        caseId={caseId}
        record={editing}
        opened={formOpened}
        onClose={formHandlers.close}
        lockedPersonId={personId}
      />
      <LifeEventDeleteModal caseId={caseId} record={deleting} opened={deleteOpened} onClose={deleteHandlers.close} />
    </>
  );
}
